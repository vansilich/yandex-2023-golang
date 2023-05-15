package courier

import (
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/internal/repository/repositories"
	validatations "yandex-team.ru/bstask/pkg/validations"
)

type CourierUseCase struct {
	validator         *validator.Validate
	CourierRepo       *repositories.CourierRepo
	OrderRepo         *repositories.OrderRepo
	DeliveryGroupRepo *repositories.DeliveryGroupRepo
}

func New(
	curstrg *repositories.CourierRepo,
	ordrepo *repositories.OrderRepo,
	dgrepo *repositories.DeliveryGroupRepo,
) *CourierUseCase {

	v := validator.New()
	v.RegisterValidation("each_HH_MM_time", validatations.Each_HH_MM_time)
	v.RegisterValidation("each_HH_MM_HH_MM_time_interval", validatations.Each_HH_MM_HH_MM_time_interval)
	v.RegisterValidation("courier_type", courier_type)

	return &CourierUseCase{
		CourierRepo:       curstrg,
		OrderRepo:         ordrepo,
		DeliveryGroupRepo: dgrepo,
		validator:         v,
	}
}

func (uc *CourierUseCase) CreateCouriers(couriers []CourierToCreateDTO) (*[]entity.Courier, error) {

	toCreate := []repositories.CourierToCreateDTO{}
	for _, c := range couriers {
		if err := uc.validator.Struct(c); err != nil {
			return nil, err
		}

		intervals := []repositories.CourierWorkingHoursIntervalDTO{}
		for _, i := range c.WorkingHours {
			spl := strings.Split(i, "-")

			startTime, err := time.Parse("15:04", spl[0])
			if err != nil {
				return nil, err
			}

			endTime, err := time.Parse("15:04", spl[1])
			if err != nil {
				return nil, err
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

	savedCouriers, err := uc.CourierRepo.BatchCreate(toCreate)
	if err != nil {
		return nil, err
	}

	return savedCouriers, nil
}

func (uc *CourierUseCase) GetById(id uint64) (*entity.Courier, error) {

	courier, err := uc.CourierRepo.FindById(id)
	if err != nil {
		return nil, err
	}

	return courier, nil
}

func (uc *CourierUseCase) PaginatedGetAll(offset, limit int32) (*[]entity.Courier, error) {

	couriers, err := uc.CourierRepo.PaginatedFetchAll(offset, limit)
	if err != nil {
		return nil, err
	}

	return couriers, nil
}

func (uc *CourierUseCase) MetaInInterval(courier *entity.Courier, startDate, endDate time.Time) (*CourierMetaDTO, error) {

	startDate = startDate.UTC()
	endDate = endDate.UTC()

	res := CourierMetaDTO{}

	ordersCost, err := uc.OrderRepo.CostInIntervalByCourierId(courier.ID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if ordersCost != nil {
		ratio, err := courier.SalaryRatio()
		if err != nil {
			return nil, err
		}

		earnings := int32(*ordersCost) * int32(ratio)
		res.Earnings = &earnings
	}

	ordersCount, err := uc.OrderRepo.CountInIntervalByCourierId(courier.ID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if ordersCount != nil {
		ratio, err := courier.RatingRatio()
		if err != nil {
			return nil, err
		}

		diff := endDate.Sub(startDate)

		if diff != 0 {
			rating := int32(*ordersCount) / int32(diff.Hours()) * int32(ratio)
			res.Rating = &rating
		}
	}

	return &res, nil
}

func (uc *CourierUseCase) Assignments(courierIDs []uint64, date time.Time) ([]AssignResponseGroupItem, error) {

	couriersOrders := make(map[uint64]AssignResponseGroupItem)
	var groups *[]entity.DeliveryGroup
	var err error

	if len(courierIDs) == 0 {
		groups, err = uc.DeliveryGroupRepo.AllByDate(date)
		if err != nil {
			return []AssignResponseGroupItem{}, nil
		}
	} else {
		groups, err = uc.DeliveryGroupRepo.AllByDateAndIds(courierIDs, date)
		if err != nil {
			return []AssignResponseGroupItem{}, err
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
			return []AssignResponseGroupItem{}, err
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
