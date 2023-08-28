package courier

import (
	"context"
	"strings"
	"time"

	"github.com/avito-tech/go-transaction-manager/trm/manager"
	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask"
	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/internal/repository/repositories"
	validatations "yandex-team.ru/bstask/pkg/validations"
)

type CourierUseCase struct {
	trm               *manager.Manager
	validator         *validator.Validate
	CourierRepo       *repositories.CourierRepo
	OrderRepo         *repositories.OrderRepo
	DeliveryGroupRepo *repositories.DeliveryGroupRepo
}

func New(
	trm *manager.Manager,
	curstrg *repositories.CourierRepo,
	ordrepo *repositories.OrderRepo,
	dgrepo *repositories.DeliveryGroupRepo,
) *CourierUseCase {

	v := validator.New()
	v.RegisterValidation("each_HH_MM_time", validatations.Each_HH_MM_time)
	v.RegisterValidation("each_HH_MM_HH_MM_time_interval", validatations.Each_HH_MM_HH_MM_time_interval)
	v.RegisterValidation("courier_type", courier_type)

	return &CourierUseCase{
		trm:               trm,
		CourierRepo:       curstrg,
		OrderRepo:         ordrepo,
		DeliveryGroupRepo: dgrepo,
		validator:         v,
	}
}

func (uc *CourierUseCase) CreateCouriers(ctx context.Context, couriers []CourierToCreateDTO) (*[]entity.Courier, error) {
	op := "usecase.courier.CreateCouriers"

	toCreate := []repositories.CourierToCreateDTO{}
	for _, c := range couriers {
		if err := uc.validator.Struct(c); err != nil {
			return nil, bstask.ErrorWithCode(bstask.OpError(op, err), bstask.EINVALID)
		}

		intervals := []repositories.CourierWorkingHoursIntervalDTO{}
		for _, i := range c.WorkingHours {
			spl := strings.Split(i, "-")

			startTime, err := time.Parse("15:04", spl[0])
			if err != nil {
				return nil, bstask.ErrorWithCode(bstask.OpError(op, err), bstask.EINVALID)
			}

			endTime, err := time.Parse("15:04", spl[1])
			if err != nil {
				return nil, bstask.ErrorWithCode(bstask.OpError(op, err), bstask.EINVALID)
			}

			intervals = append(intervals, repositories.CourierWorkingHoursIntervalDTO{
				StartTime: startTime,
				EndTime:   endTime,
			})
		}

		toCreate = append(toCreate, repositories.CourierToCreateDTO{
			CourierType:  c.CourierType,
			Regions:      c.Regions,
			WorkingHours: intervals,
		})
	}

	var savedCouriers *[]entity.Courier
	var err error

	err = uc.trm.Do(ctx, func(ctx context.Context) error {
		savedCouriers, err = uc.CourierRepo.BatchCreate(ctx, toCreate)
		return err
	})
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return savedCouriers, nil
}

func (uc *CourierUseCase) GetById(ctx context.Context, id uint64) (*entity.Courier, error) {
	op := "usecase.courier.GetById"

	courier, err := uc.CourierRepo.FindById(ctx, id)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return courier, nil
}

func (uc *CourierUseCase) PaginatedGetAll(ctx context.Context, offset, limit int32) (*[]entity.Courier, error) {
	op := "usecase.courier.PaginatedGetAll"

	couriers, err := uc.CourierRepo.PaginatedFetchAll(ctx, offset, limit)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return couriers, nil
}

func (uc *CourierUseCase) MetaInInterval(ctx context.Context, courier *entity.Courier, startDate, endDate time.Time) (*CourierMetaDTO, error) {
	op := "usecase.courier.MetaInInterval"

	startDate = startDate.UTC()
	endDate = endDate.UTC()

	if startDate.After(endDate) {
		return nil, &bstask.Error{
			Op:      op,
			Code:    bstask.EINVALID,
			Message: ":startDate is after :endDate param",
		}
	}

	res := CourierMetaDTO{}

	ordersCount, err := uc.OrderRepo.CountInIntervalByCourierId(ctx, courier.ID, startDate, endDate)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}
	if ordersCount == 0 {
		return &CourierMetaDTO{}, nil
	}

	ratingRatio, err := courier.RatingRatio()
	if err != nil {
		return nil, bstask.OpError(op, err)
	}
	if ratingRatio == 0 {
		return nil, &bstask.Error{
			Op:      op,
			Code:    bstask.EINTERNAL,
			Message: "rating ratio of courier is 0",
			Fields: map[string]interface{}{
				"courier_id":   courier.ID,
				"courier_type": courier.CourierType,
			},
		}
	}

	diff := endDate.Sub(startDate)
	if diff != 0 {
		rating := int32(ordersCount) / int32(diff.Hours()) * int32(ratingRatio)
		res.Rating = &rating
	}

	ordersCost, err := uc.OrderRepo.CostInIntervalByCourierId(ctx, courier.ID, startDate, endDate)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	salaryRatio, err := courier.SalaryRatio()
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	earnings := int32(ordersCost) * int32(salaryRatio)
	res.Earnings = &earnings

	return &res, nil
}

func (uc *CourierUseCase) Assignments(ctx context.Context, courierIDs []uint64, date time.Time) ([]AssignResponseGroupItem, error) {
	op := "usecase.courier.Assignments"

	couriersOrders := make(map[uint64]AssignResponseGroupItem)
	var groups *[]entity.DeliveryGroup
	var err error

	if len(courierIDs) == 0 {
		groups, err = uc.DeliveryGroupRepo.AllByDate(ctx, date)
		if err != nil {
			return []AssignResponseGroupItem{}, bstask.OpError(op, err)
		}
	} else {
		groups, err = uc.DeliveryGroupRepo.AllByDateAndIds(ctx, courierIDs, date)
		if err != nil {
			return []AssignResponseGroupItem{}, bstask.OpError(op, err)
		}
	}

	res := []AssignResponseGroupItem{}
	for _, g := range *groups {

		assignResponseGroupItem, ok := couriersOrders[g.CourierID]
		if !ok {
			assignResponseGroupItem = AssignResponseGroupItem{
				CourierId: g.CourierID,
				Orders:    make(map[uint64]AssignOrdersGroup),
			}
		}

		orders, err := uc.OrderRepo.OrdersInGroup(g.ID)
		if err != nil {
			return []AssignResponseGroupItem{}, bstask.OpError(op, err)
		}

		for _, order := range *orders {
			assignOrdersGroup, ok := assignResponseGroupItem.Orders[g.ID]
			if !ok {
				assignOrdersGroup = AssignOrdersGroup{
					GroupOrderId: g.ID,
					Orders:       []entity.Order{},
				}
			}

			assignOrdersGroup.Orders = append(assignOrdersGroup.Orders, order)

			assignResponseGroupItem.Orders[g.ID] = assignOrdersGroup
		}

		couriersOrders[g.CourierID] = assignResponseGroupItem
	}

	for _, gi := range couriersOrders {
		res = append(res, gi)
	}

	return res, nil
}
