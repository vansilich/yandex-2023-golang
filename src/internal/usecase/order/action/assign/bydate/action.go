package bydate

import (
	"context"
	"time"

	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/internal/repository/repositories"
)

var couriersOrders map[uint64]AssignResponseGroupItem = make(map[uint64]AssignResponseGroupItem)

type ActionAssignByDate struct {
	CourierRepo       *repositories.CourierRepo
	OrderRepo         *repositories.OrderRepo
	DeliveryGroupRepo *repositories.DeliveryGroupRepo
}

func New(
	CourierRepo *repositories.CourierRepo,
	OrderRepo *repositories.OrderRepo,
	DeliveryGroupRepo *repositories.DeliveryGroupRepo,
) *ActionAssignByDate {
	return &ActionAssignByDate{
		CourierRepo:       CourierRepo,
		OrderRepo:         OrderRepo,
		DeliveryGroupRepo: DeliveryGroupRepo,
	}
}

func (a *ActionAssignByDate) Assign(ctx context.Context, assignDate time.Time) (AssignResponseGroup, error) {

	assignDate = assignDate.UTC()

	res := AssignResponseGroup{
		Date: assignDate,
	}

	footCouriersWorkingHours, err := a.CourierRepo.AllWorkingHoursByCourierType(ctx, entity.FOOT)
	if err != nil {
		return AssignResponseGroup{}, err
	}

	for _, wh := range *footCouriersWorkingHours {
		err := a.assignToWorkingInterval(ctx, assignDate, wh, *a.OrderRepo)
		if err != nil {
			return AssignResponseGroup{}, err
		}
	}

	bikeCouriersWorkingHours, err := a.CourierRepo.AllWorkingHoursByCourierType(ctx, entity.BIKE)
	if err != nil {
		return AssignResponseGroup{}, err
	}

	for _, wh := range *bikeCouriersWorkingHours {
		err := a.assignToWorkingInterval(ctx, assignDate, wh, *a.OrderRepo)
		if err != nil {
			return AssignResponseGroup{}, err
		}
	}

	autoCouriersWorkingHours, err := a.CourierRepo.AllWorkingHoursByCourierType(ctx, entity.AUTO)
	if err != nil {
		return AssignResponseGroup{}, err
	}

	for _, wh := range *autoCouriersWorkingHours {
		err := a.assignToWorkingInterval(ctx, assignDate, wh, *a.OrderRepo)
		if err != nil {
			return AssignResponseGroup{}, err
		}
	}

	for _, gr := range couriersOrders {
		res.Couriers = append(res.Couriers, gr)
	}

	return res, nil
}

func (a *ActionAssignByDate) assignToWorkingInterval(
	ctx context.Context,
	assignDate time.Time,
	wh repositories.AllWorkingHoursRes,
	orderRepo repositories.OrderRepo,
) error {

	startDateTime := time.Date(assignDate.Year(), assignDate.Month(), assignDate.Day(), wh.StartTime.Hour(), wh.StartTime.Minute(), wh.StartTime.Second(), 0, assignDate.Location())
	endDateTime := time.Date(assignDate.Year(), assignDate.Month(), assignDate.Day(), wh.EndTime.Hour(), wh.EndTime.Minute(), wh.EndTime.Second(), 0, assignDate.Location())

	potential, err := entity.DeliveryPotentialForType(entity.CourierType(wh.CourierType))
	if err != nil {
		return err
	}

	courierState, err := initCourierState(
		a.DeliveryGroupRepo,
		wh.CourierID,
		wh.WorkingHoursID,
		potential,
		entity.CourierType(wh.CourierType),
		wh.Regions,
		startDateTime,
		endDateTime,
	)
	if err != nil {
		return nil
	}

	for {
		var order *entity.Order

		if courierState.isTimeToFlush() {
			courierState.flush(ctx)
		}

		if courierState.isTimeToStop() {
			break
		}

		order, err := a.orderForCurrentState(ctx, orderRepo, *courierState)
		if err != nil {
			return err
		}

		if order == nil {
			if courierState.isOnTheWay {
				courierState.flush(ctx)
				order, err = a.orderForCurrentState(ctx, orderRepo, *courierState)
				if err != nil {
					return err
				}
			}

			if order == nil {
				break
			}
		}

		completeDateTime, discountPrice, err := courierState.addOrder(ctx, *order)
		if err != nil {
			return nil
		}

		err = orderRepo.SetCompletedInfo(ctx, order, repositories.OrderCompleteInfoDTO{
			CourierID:       wh.CourierID,
			DeliveryGroupID: courierState.deliveryGroup.ID,
			Cost:            discountPrice,
			CompleteTime:    completeDateTime,
		})
		if err != nil {
			return err
		}

		a.saveForResponse(*courierState, wh, *order)
	}

	return nil
}

func (a *ActionAssignByDate) orderForCurrentState(
	ctx context.Context,
	orderRepo repositories.OrderRepo,
	cs courierBatchState,
) (*entity.Order, error) {

	var (
		order *entity.Order
		err   error
	)

	params := repositories.FindInRegionsForCourierDTO{
		MaxWeight:          cs.availableWeight(),
		Regions:            []int32{cs.currRegion},
		DeliveryHoursStart: cs.nextDeliveryStartDateTime,
		DeliveryHoursEnd:   cs.shiftEndDateTime,
		OrderByWeightASC:   false,
		WithGap:            false,
	}

	if cs.isOnTheWay {
		// Search in specific region
		if cs.nextWillBeLast() {
			order, err = orderRepo.FindInRegionForCourier(ctx, params)
			if err != nil {
				return nil, err
			}
		} else {
			params.OrderByWeightASC = true
			order, err = orderRepo.FindInRegionForCourier(ctx, params)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// Search in any available region
		params.Regions = cs.availableRegions
		params.OrderByWeightASC = true

		order, err = orderRepo.FindInRegionForCourier(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	if order == nil {
		// try find with gap
		params.WithGap = true
		params.OrderByWeightASC = true

		order, err = orderRepo.FindInRegionForCourier(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	return order, nil
}

func (a *ActionAssignByDate) saveForResponse(
	courierState courierBatchState,
	wh repositories.AllWorkingHoursRes,
	order entity.Order,
) {

	assignedGroups, ok := couriersOrders[wh.CourierID]
	if !ok {
		gi := AssignResponseGroupItem{
			CourierId: wh.CourierID,
			Orders:    make(map[uint64]AssignOrdersGroup),
		}
		couriersOrders[wh.CourierID] = gi
		assignedGroups = gi
	}

	assignedOrdersGroup, ok := assignedGroups.Orders[courierState.deliveryGroup.ID]
	if !ok {
		og := AssignOrdersGroup{
			GroupOrderId: courierState.deliveryGroup.ID,
			Orders:       []entity.Order{},
		}

		assignedGroups.Orders[courierState.deliveryGroup.ID] = og
		assignedOrdersGroup = og
	}

	assignedOrdersGroup.Orders = append(assignedOrdersGroup.Orders, order)

	assignedGroups.Orders[courierState.deliveryGroup.ID] = assignedOrdersGroup
}
