package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"yandex-team.ru/bstask/internal/entity"
	appErrors "yandex-team.ru/bstask/internal/errors"
	"yandex-team.ru/bstask/pkg/gorm/types"
)

var (
	OrderNotFoundError     = appErrors.NewInternalError(nil, "Order not found", true)
	CourierAlreadyAssigned = appErrors.NewInternalError(nil, "Courier already assigned to order", true)
)

// @migration
type Order struct {
	ID              uint64 `gorm:"primaryKey"`
	Weight          float64
	Regions         int32
	DeliveryHours   []OrderDeliveryHours `gorm:"foreignKey:OrderID;references:ID"`
	Cost            uint32
	CompletedTime   *time.Time
	DeliveryGroupID *uint64
	DeliveryGroup   *DeliveryGroup `gorm:"foreignKey:DeliveryGroupID"`
}

// @migration
type OrderDeliveryHours struct {
	ID        uint64 `gorm:"primaryKey"`
	OrderID   uint64
	Order     *Order `gorm:"foreignKey:OrderID"`
	StartTime types.Time
	EndTime   types.Time
}

type OrderRepo struct {
	gorm *gorm.DB
}

func NewOrderRepo(grm *gorm.DB) *OrderRepo {
	return &OrderRepo{
		gorm: grm,
	}
}

func (s *OrderRepo) Atomic(fn func(repo OrderRepo) error) error {
	tx := s.gorm.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	newRepo := OrderRepo{gorm: tx}
	err := fn(newRepo)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

type OrderToCreateDTO struct {
	Weight        float64
	Regions       int32
	DeliveryHours []OrderDeliveryHoursIntervalDTO
	Cost          uint32
}

type OrderDeliveryHoursIntervalDTO struct {
	StartTime time.Time
	EndTime   time.Time
}

func (s *OrderRepo) BatchCreate(newOrders []OrderToCreateDTO) (*[]entity.Order, error) {

	orders := []Order{}

	err := s.Atomic(func(repo OrderRepo) error {
		for _, o := range newOrders {

			// BUG: if we assign `delivery hours` to courier here,
			// we will deal with duplicate INSERT queries

			orders = append(orders, Order{
				Weight:  o.Weight,
				Regions: o.Regions,
				Cost:    o.Cost,
			})
		}

		dbRes := repo.gorm.CreateInBatches(orders, 20)
		if dbRes.Error != nil {
			return dbRes.Error
		}

		for i, c := range newOrders {
			order := &orders[i]

			for _, dh := range c.DeliveryHours {
				(*order).DeliveryHours = append(order.DeliveryHours, OrderDeliveryHours{
					Order:     order,
					StartTime: types.NewTime(dh.StartTime.Hour(), dh.StartTime.Minute(), dh.StartTime.Second()),
					EndTime:   types.NewTime(dh.EndTime.Hour(), dh.EndTime.Minute(), dh.EndTime.Second()),
				})
			}

			dbRes = repo.gorm.Save(order)
			if dbRes.Error != nil {
				return dbRes.Error
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	res := []entity.Order{}
	for _, o := range orders {

		dh := []entity.OrderDeliveryHours{}
		for _, t := range o.DeliveryHours {
			dh = append(dh, entity.OrderDeliveryHours{
				ID:        t.ID,
				StartTime: time.Time(t.StartTime),
				EndTime:   time.Time(t.EndTime),
			})
		}

		res = append(res, entity.Order{
			ID:              o.ID,
			Weight:          o.Weight,
			Regions:         o.Regions,
			DeliveryHours:   dh,
			Cost:            o.Cost,
			CompletedTime:   o.CompletedTime,
			DeliveryGroupID: o.DeliveryGroupID,
		})
	}

	return &res, nil
}

func (s *OrderRepo) FindById(id uint64) (*entity.Order, error) {

	var order Order

	err := s.gorm.Model(&Order{}).Preload("DeliveryHours").Find(&order, int(id)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {

			OrderNotFoundError.Err = err
			return nil, OrderNotFoundError
		}

		return nil, err
	}

	dh := []entity.OrderDeliveryHours{}
	for _, t := range order.DeliveryHours {
		dh = append(dh, entity.OrderDeliveryHours{
			ID:        t.ID,
			StartTime: time.Time(t.StartTime),
			EndTime:   time.Time(t.EndTime),
		})
	}

	return &entity.Order{
		ID:              order.ID,
		Weight:          order.Weight,
		Regions:         order.Regions,
		DeliveryHours:   dh,
		Cost:            order.Cost,
		CompletedTime:   order.CompletedTime,
		DeliveryGroupID: order.DeliveryGroupID,
	}, nil
}

func (s *OrderRepo) PaginatedFetchAll(offset, limit int32) (*[]entity.Order, error) {

	orders := []Order{}

	err := s.gorm.Model(&Order{}).Preload("DeliveryHours").Limit(int(limit)).Offset(int(offset)).Find(&orders).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Order{}
	for _, o := range orders {

		dh := []entity.OrderDeliveryHours{}
		for _, t := range o.DeliveryHours {
			dh = append(dh, entity.OrderDeliveryHours{
				ID:        t.ID,
				StartTime: time.Time(t.StartTime),
				EndTime:   time.Time(t.EndTime),
			})
		}

		res = append(res, entity.Order{
			ID:              o.ID,
			Weight:          o.Weight,
			Regions:         o.Regions,
			DeliveryHours:   dh,
			Cost:            o.Cost,
			CompletedTime:   o.CompletedTime,
			DeliveryGroupID: o.DeliveryGroupID,
		})
	}

	return &res, nil
}

type OrderCompleteInfoDTO struct {
	CourierID       uint64
	DeliveryGroupID uint64
	Cost            uint32
	CompleteTime    time.Time
}

func (s *OrderRepo) SetCompletedInfo(order *entity.Order, info OrderCompleteInfoDTO) error {

	order.Cost = info.Cost
	order.CompletedTime = &info.CompleteTime
	order.DeliveryGroupID = &info.DeliveryGroupID

	dh := []OrderDeliveryHours{}
	for _, t := range order.DeliveryHours {
		dh = append(dh, OrderDeliveryHours{
			ID:        t.ID,
			StartTime: types.Time(t.StartTime),
			EndTime:   types.Time(t.EndTime),
		})
	}

	o := Order{
		ID:              order.ID,
		Weight:          order.Weight,
		Regions:         order.Regions,
		DeliveryHours:   dh,
		Cost:            info.Cost,
		CompletedTime:   &info.CompleteTime,
		DeliveryGroupID: &info.DeliveryGroupID,
	}

	err := s.gorm.Save(&o).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderRepo) CostInIntervalByCourierId(
	courierID uint64,
	startDate,
	endDate time.Time,
) (*uint64, error) {

	var cost *uint64 = nil

	err := s.gorm.Raw(`
		SELECT SUM(o."cost") as "cost" FROM "orders" as o
		LEFT JOIN "delivery_groups" as odg ON odg."id" = o."delivery_group_id"
		WHERE odg."courier_id" = ?
			AND o."completed_time" between ? and ?`,
		courierID,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	).Scan(&cost).Error

	if err != nil {
		return nil, err
	}

	return cost, nil
}

func (s *OrderRepo) CountInIntervalByCourierId(
	courierID uint64,
	startDate,
	endDate time.Time,
) (*uint64, error) {

	var count *uint64 = nil

	err := s.gorm.Raw(`
		SELECT COUNT(o.*) as "count" FROM "orders" as o
		LEFT JOIN "delivery_groups" as odg ON odg."id" = o."delivery_group_id"
		WHERE odg."courier_id" = ?
			AND o."completed_time" between ? and ?`,
		courierID,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	).Row().Scan(&count)

	if err != nil {
		return nil, err
	}

	return count, nil
}

type FindInRegionsForCourierDTO struct {
	MaxWeight          float64
	Regions            []int32
	DeliveryHoursStart time.Time
	DeliveryHoursEnd   time.Time
}

func (s *OrderRepo) FindInRegionForCourier(
	params FindInRegionsForCourierDTO,
	orderByWeightASC bool,
) (*entity.Order, error) {

	var order *Order = nil

	regionsArr, err := pq.Int32Array(params.Regions).Value()
	if err != nil {
		return nil, err
	}

	var orderingType string
	if orderByWeightASC {
		orderingType = "ASC"
	} else {
		orderingType = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT "o".* FROM "orders" as "o"
		LEFT JOIN "order_delivery_hours" "odh"
			ON "odh"."order_id" = "o"."id"
		WHERE "o"."delivery_group_id" IS NULL
			AND "o"."weight" <= ?
			AND "o"."regions" = ANY(?)
			AND "odh"."start_time" <= ?
			AND "odh"."end_time" >= ?
		ORDER BY "o"."weight" %s
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, orderingType)

	err = s.gorm.Raw(
		query,
		params.MaxWeight,
		regionsArr,
		params.DeliveryHoursStart.Format("15:04:05"),
		params.DeliveryHoursEnd.Format("15:04:05"),
	).Scan(&order).Error

	if err != nil {
		return nil, err
	}

	if order != nil {

		var deliveryHours []OrderDeliveryHours
		err := s.gorm.Where("order_id = ?", order.ID).Find(&deliveryHours).Error
		if err != nil {
			return nil, err
		}
		order.DeliveryHours = deliveryHours

		dh := []entity.OrderDeliveryHours{}
		for _, t := range order.DeliveryHours {
			dh = append(dh, entity.OrderDeliveryHours{
				ID:        t.ID,
				StartTime: time.Time(t.StartTime),
				EndTime:   time.Time(t.EndTime),
			})
		}

		res := entity.Order{
			ID:            order.ID,
			Weight:        order.Weight,
			Regions:       order.Regions,
			DeliveryHours: dh,
			Cost:          order.Cost,
			CompletedTime: order.CompletedTime,
		}

		return &res, nil
	}

	return nil, nil
}

func (s *OrderRepo) FindInRegionWithGapForCourier(params FindInRegionsForCourierDTO) (*entity.Order, error) {

	var order *Order = nil

	regionsArr, err := pq.Int32Array(params.Regions).Value()
	if err != nil {
		return nil, err
	}

	err = s.gorm.Raw(`SELECT "o".* FROM "orders" as "o"
		LEFT JOIN "order_delivery_hours" "odh"
			ON "odh"."order_id" = "o"."id"
		WHERE "o"."delivery_group_id" IS NULL
			AND "o"."weight" <= ?
			AND "o"."regions" = ANY(?)
			AND "odh"."start_time" >= ?
			AND "odh"."end_time" >= ?
		ORDER BY "o"."weight" ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`,
		params.MaxWeight,
		regionsArr,
		params.DeliveryHoursStart.Format("15:04:05"),
		params.DeliveryHoursEnd.Format("15:04:05"),
	).Scan(&order).Error

	if err != nil {
		return nil, err
	}

	if order != nil {

		var deliveryHours []OrderDeliveryHours
		err := s.gorm.Where("order_id = ?", order.ID).Find(&deliveryHours).Error
		if err != nil {
			return nil, err
		}
		order.DeliveryHours = deliveryHours

		dh := []entity.OrderDeliveryHours{}
		for _, t := range order.DeliveryHours {
			dh = append(dh, entity.OrderDeliveryHours{
				ID:        t.ID,
				StartTime: time.Time(t.StartTime),
				EndTime:   time.Time(t.EndTime),
			})
		}

		res := entity.Order{
			ID:            order.ID,
			Weight:        order.Weight,
			Regions:       order.Regions,
			DeliveryHours: dh,
			Cost:          order.Cost,
			CompletedTime: order.CompletedTime,
		}

		return &res, nil
	}

	return nil, nil
}

func (s *OrderRepo) OrdersInGroup(groupID uint64) (*[]entity.Order, error) {
	orders := []Order{}

	err := s.gorm.Where(&Order{DeliveryGroupID: &groupID}).Preload("DeliveryHours").Find(&orders).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Order{}
	for _, o := range orders {

		dh := []entity.OrderDeliveryHours{}
		for _, t := range o.DeliveryHours {
			dh = append(dh, entity.OrderDeliveryHours{
				ID:        t.ID,
				StartTime: time.Time(t.StartTime),
				EndTime:   time.Time(t.EndTime),
			})
		}

		res = append(res, entity.Order{
			ID:              o.ID,
			Weight:          o.Weight,
			Regions:         o.Regions,
			DeliveryHours:   dh,
			Cost:            o.Cost,
			CompletedTime:   o.CompletedTime,
			DeliveryGroupID: o.DeliveryGroupID,
		})
	}

	return &res, nil
}
