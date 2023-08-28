package order

import (
	"context"
	"strings"
	"time"

	"github.com/avito-tech/go-transaction-manager/trm/manager"
	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask"
	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/internal/repository/repositories"
	"yandex-team.ru/bstask/internal/usecase/order/action/assign/bydate"
	validatations "yandex-team.ru/bstask/pkg/validations"
)

type OrderUseCase struct {
	trm               *manager.Manager
	validator         *validator.Validate
	OrderRepo         *repositories.OrderRepo
	CourierRepo       *repositories.CourierRepo
	DeliveryGroupRepo *repositories.DeliveryGroupRepo
}

func New(
	trm *manager.Manager,
	ordrepo *repositories.OrderRepo,
	courrepo *repositories.CourierRepo,
	ogrepo *repositories.DeliveryGroupRepo,
) *OrderUseCase {

	v := validator.New()
	v.RegisterValidation("each_HH_MM_HH_MM_time_interval", validatations.Each_HH_MM_HH_MM_time_interval)

	return &OrderUseCase{
		trm:               trm,
		OrderRepo:         ordrepo,
		CourierRepo:       courrepo,
		DeliveryGroupRepo: ogrepo,
		validator:         v,
	}
}

func (uc *OrderUseCase) CreateOrders(ctx context.Context, orders []OrderToCreateDTO) (*[]entity.Order, error) {
	op := "OrderUseCase.CreateOrders"

	toCreate := []repositories.OrderToCreateDTO{}
	for _, c := range orders {
		if err := uc.validator.Struct(c); err != nil {
			return nil, bstask.OpError(op, err)
		}

		intervals := []repositories.OrderDeliveryHoursIntervalDTO{}
		for _, i := range c.DeliveryHours {
			spl := strings.Split(i, "-")

			startTime, err := time.Parse("15:04", spl[0])
			if err != nil {
				return nil, bstask.OpError(op, err)
			}

			endTime, err := time.Parse("15:04", spl[1])
			if err != nil {
				return nil, bstask.OpError(op, err)
			}

			intervals = append(intervals, repositories.OrderDeliveryHoursIntervalDTO{
				StartTime: startTime,
				EndTime:   endTime,
			})
		}

		toCreate = append(toCreate, repositories.OrderToCreateDTO{
			Weight:        c.Weight,
			Regions:       c.Regions,
			DeliveryHours: intervals,
			Cost:          c.Cost,
		})
	}

	var savedOrders *[]entity.Order
	var err error
	err = uc.trm.Do(ctx, func(ctx context.Context) error {
		savedOrders, err = uc.OrderRepo.BatchCreate(ctx, toCreate)
		return err
	})
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return savedOrders, nil
}

func (uc *OrderUseCase) GetById(ctx context.Context, id uint64) (*entity.Order, error) {
	const op = "OrderUseCase.GetById"

	order, err := uc.OrderRepo.FindById(ctx, id)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return order, nil
}

func (uc *OrderUseCase) PaginatedGetAll(ctx context.Context, offset, limit int32) (*[]entity.Order, error) {
	op := "OrderUseCase.PaginatedGetAll"

	couriers, err := uc.OrderRepo.PaginatedFetchAll(ctx, offset, limit)
	if err != nil {
		return nil, bstask.OpError(op, err)
	}

	return couriers, nil
}

func (uc *OrderUseCase) Complete(ctx context.Context, toComplete []OrderToCompleteDTO) (*[]entity.Order, error) {
	const op = "OrderUseCase.Complete"

	res := []entity.Order{}

	err := uc.trm.Do(ctx, func(ctx context.Context) error {
		for _, i := range toComplete {
			if err := uc.validator.Struct(i); err != nil {
				return &bstask.Error{Op: op, Err: err, Code: bstask.EINVALID}
			}

			courierEntity, err := uc.CourierRepo.FindById(ctx, uint64(i.CourierId))
			if err != nil {
				return bstask.OpError(op, err)
			}

			orderEntity, err := uc.OrderRepo.FindById(ctx, uint64(i.OrderId))
			if err != nil {
				return bstask.OpError(op, err)
			}

			wh, err := uc.CourierRepo.WorkingIntervalForDelivery(ctx, courierEntity.ID, i.CompleteTime, i.CompleteTime)
			if err != nil {
				return bstask.OpError(op, err)
			}

			duration, err := entity.NextDeliveryTimeInRegion(courierEntity.CourierType, 0)
			if err != nil {
				return bstask.OpError(op, err)
			}

			deliveryGroupEntity, err := uc.DeliveryGroupRepo.CreateGroup(
				ctx,
				courierEntity.ID,
				wh.ID,
				i.CompleteTime,
				i.CompleteTime.Add(-duration),
				i.CompleteTime,
			)
			if err != nil {
				return bstask.OpError(op, err)
			}

			if orderEntity.DeliveryGroupID != nil && *orderEntity.DeliveryGroupID != uint64(deliveryGroupEntity.ID) {
				return &bstask.Error{
					Op:      op,
					Message: "courier already assigned to order",
					Fields: map[string]interface{}{
						"courier_id": courierEntity.ID,
						"order_id":   orderEntity.ID,
					},
				}
			}

			err = uc.OrderRepo.SetCompletedInfo(ctx, orderEntity, repositories.OrderCompleteInfoDTO{
				CourierID:       courierEntity.ID,
				DeliveryGroupID: deliveryGroupEntity.ID,
				Cost:            orderEntity.Cost,
				CompleteTime:    i.CompleteTime.UTC(),
			})

			if err != nil {
				return bstask.OpError(op, err)
			}

			res = append(res, *orderEntity)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (uc *OrderUseCase) AssignByDate(ctx context.Context, assignDate time.Time) (bydate.AssignResponseGroup, error) {
	action := bydate.New(uc.CourierRepo, uc.OrderRepo, uc.DeliveryGroupRepo)

	var res bydate.AssignResponseGroup
	var err error
	err = uc.trm.Do(ctx, func(ctx context.Context) error {
		res, err = action.Assign(ctx, assignDate)
		return err
	})

	return res, err
}
