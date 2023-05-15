package order

import (
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask/internal/entity"
	appErrors "yandex-team.ru/bstask/internal/errors"
	"yandex-team.ru/bstask/internal/repository/repositories"
	"yandex-team.ru/bstask/internal/usecase/order/action/assign/bydate"
	validatations "yandex-team.ru/bstask/pkg/validations"
)

type OrderUseCase struct {
	validator         *validator.Validate
	OrderRepo         *repositories.OrderRepo
	CourierRepo       *repositories.CourierRepo
	DeliveryGroupRepo *repositories.DeliveryGroupRepo
}

func New(
	ordrepo *repositories.OrderRepo,
	courrepo *repositories.CourierRepo,
	ogrepo *repositories.DeliveryGroupRepo,
) *OrderUseCase {

	v := validator.New()
	v.RegisterValidation("each_HH_MM_HH_MM_time_interval", validatations.Each_HH_MM_HH_MM_time_interval)

	return &OrderUseCase{
		OrderRepo:         ordrepo,
		CourierRepo:       courrepo,
		DeliveryGroupRepo: ogrepo,
		validator:         v,
	}
}

func (uc *OrderUseCase) CreateOrders(orders []OrderToCreateDTO) (*[]entity.Order, error) {

	toCreate := []repositories.OrderToCreateDTO{}
	for _, c := range orders {
		if err := uc.validator.Struct(c); err != nil {
			return nil, err
		}

		intervals := []repositories.OrderDeliveryHoursIntervalDTO{}
		for _, i := range c.DeliveryHours {
			spl := strings.Split(i, "-")

			startTime, err := time.Parse("15:04", spl[0])
			if err != nil {
				return nil, err
			}

			endTime, err := time.Parse("15:04", spl[1])
			if err != nil {
				return nil, err
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

	savedOrders, err := uc.OrderRepo.BatchCreate(toCreate)
	if err != nil {
		return nil, err
	}

	return savedOrders, nil
}

func (uc *OrderUseCase) GetById(id uint64) (*entity.Order, error) {

	order, err := uc.OrderRepo.FindById(id)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (uc *OrderUseCase) PaginatedGetAll(offset, limit int32) (*[]entity.Order, error) {
	couriers, err := uc.OrderRepo.PaginatedFetchAll(offset, limit)
	if err != nil {
		return nil, err
	}

	return couriers, nil
}

func (uc *OrderUseCase) Complete(toComplete []OrderToCompleteDTO) (*[]entity.Order, error) {

	res := []entity.Order{}

	err := uc.OrderRepo.Atomic(func(repo repositories.OrderRepo) error {

		for _, i := range toComplete {
			if err := uc.validator.Struct(i); err != nil {
				return err
			}

			courierEntity, err := uc.CourierRepo.FindById(uint64(i.CourierId))
			if err != nil {
				return err
			}

			orderEntity, err := repo.FindById(uint64(i.OrderId))
			if err != nil {
				return err
			}

			wh, err := uc.CourierRepo.WorkingIntervalForDelivery(courierEntity.ID, i.CompleteTime, i.CompleteTime)
			if err != nil {
				return err
			}

			duration, err := entity.NextDeliveryTimeInRegion(courierEntity.CourierType, 0)
			if err != nil {
				return err
			}

			deliveryGroupEntity, err := uc.DeliveryGroupRepo.GetOrCreateGroup(
				courierEntity.ID,
				wh.ID,
				i.CompleteTime,
				i.CompleteTime.Add(-duration),
				i.CompleteTime,
			)
			if err != nil {
				return err
			}

			if orderEntity.DeliveryGroupID != nil && *orderEntity.DeliveryGroupID != uint64(deliveryGroupEntity.ID) {
				return appErrors.NewInternalError(nil, "Courier already assigned to order", true)
			}

			err = repo.SetCompletedInfo(orderEntity, repositories.OrderCompleteInfoDTO{
				CourierID:       courierEntity.ID,
				DeliveryGroupID: deliveryGroupEntity.ID,
				Cost:            orderEntity.Cost,
				CompleteTime:    i.CompleteTime.UTC(),
			})

			if err != nil {
				return err
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

func (uc *OrderUseCase) AssignByDate(assignDate time.Time) (bydate.AssignResponseGroup, error) {
	action := bydate.New(uc.CourierRepo, uc.OrderRepo, uc.DeliveryGroupRepo)

	return action.Assign(assignDate)
}
