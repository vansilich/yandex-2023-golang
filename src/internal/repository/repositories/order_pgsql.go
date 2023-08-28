package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"yandex-team.ru/bstask"
	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/pkg/gorm/types"
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
	gorm      *gorm.DB
	ctxGetter *trmgorm.CtxGetter
}

func NewOrderRepo(grm *gorm.DB, c *trmgorm.CtxGetter) *OrderRepo {
	return &OrderRepo{
		gorm:      grm,
		ctxGetter: c,
	}
}

func toOrderEntity(o Order) entity.Order {

	dh := []entity.OrderDeliveryHours{}
	for _, t := range o.DeliveryHours {
		dh = append(dh, entity.OrderDeliveryHours{
			ID:        t.ID,
			StartTime: time.Time(t.StartTime),
			EndTime:   time.Time(t.EndTime),
		})
	}

	return entity.Order{
		ID:              o.ID,
		Weight:          o.Weight,
		Regions:         o.Regions,
		DeliveryHours:   dh,
		Cost:            o.Cost,
		CompletedTime:   o.CompletedTime,
		DeliveryGroupID: o.DeliveryGroupID,
	}
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

func (s *OrderRepo) BatchCreate(ctx context.Context, newOrders []OrderToCreateDTO) (*[]entity.Order, error) {

	orders := []Order{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	for _, o := range newOrders {

		// BUG: if we assign `delivery hours` to courier here,
		// we will deal with duplicate INSERT queries

		orders = append(orders, Order{
			Weight:  o.Weight,
			Regions: o.Regions,
			Cost:    o.Cost,
		})
	}

	dbRes := db.CreateInBatches(orders, 20)
	if dbRes.Error != nil {
		return nil, dbRes.Error
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

		dbRes = db.Save(order)
		if dbRes.Error != nil {
			return nil, dbRes.Error
		}
	}

	res := []entity.Order{}
	for _, o := range orders {
		res = append(res, toOrderEntity(o))
	}

	return &res, nil
}

func (s *OrderRepo) FindById(ctx context.Context, id uint64) (*entity.Order, error) {

	var order Order

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Model(&Order{}).Preload("DeliveryHours").First(&order, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &bstask.Error{
				Op:  "repositories.OrderRepo.FindById",
				Err: err,
				Fields: map[string]interface{}{
					"order_id": id,
				},
			}
		}

		return nil, err
	}

	res := toOrderEntity(order)

	return &res, nil
}

func (s *OrderRepo) PaginatedFetchAll(ctx context.Context, offset, limit int32) (*[]entity.Order, error) {

	orders := []Order{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Model(&Order{}).Preload("DeliveryHours").Limit(int(limit)).Offset(int(offset)).Find(&orders).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Order{}
	for _, o := range orders {
		res = append(res, toOrderEntity(o))
	}

	return &res, nil
}

type OrderCompleteInfoDTO struct {
	CourierID       uint64
	DeliveryGroupID uint64
	Cost            uint32
	CompleteTime    time.Time
}

func (s *OrderRepo) SetCompletedInfo(ctx context.Context, order *entity.Order, info OrderCompleteInfoDTO) error {

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

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Save(&o).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderRepo) CostInIntervalByCourierId(
	ctx context.Context,
	courierID uint64,
	startDate,
	endDate time.Time,
) (uint64, error) {

	var cost *uint64 = nil

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Raw(`
		SELECT SUM(o."cost") as "cost" FROM "orders" as o
		LEFT JOIN "delivery_groups" as odg ON odg."id" = o."delivery_group_id"
		WHERE odg."courier_id" = ?
			AND o."completed_time" between ? and ?`,
		courierID,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	).Scan(&cost).Error

	if err != nil {
		return 0, err
	}

	return *cost, nil
}

func (s *OrderRepo) CountInIntervalByCourierId(
	ctx context.Context,
	courierID uint64,
	startDate,
	endDate time.Time,
) (uint64, error) {

	var count *uint64 = nil

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Raw(`
		SELECT COUNT(o.*) as "count" FROM "orders" as o
		LEFT JOIN "delivery_groups" as odg ON odg."id" = o."delivery_group_id"
		WHERE odg."courier_id" = ?
			AND o."completed_time" between ? and ?`,
		courierID,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	).Row().Scan(&count)

	if err != nil {
		return 0, err
	}

	return *count, nil
}

type FindInRegionsForCourierDTO struct {
	MaxWeight          float64
	Regions            []int32
	DeliveryHoursStart time.Time
	DeliveryHoursEnd   time.Time
	OrderByWeightASC   bool
	WithGap            bool
}

func (s *OrderRepo) FindInRegionForCourier(ctx context.Context, params FindInRegionsForCourierDTO) (*entity.Order, error) {

	regionsArr, err := pq.Int32Array(params.Regions).Value()
	if err != nil {
		return nil, err
	}

	var order *Order = nil
	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)

	query := `
		SELECT "o".* FROM "orders" as "o"
		LEFT JOIN "order_delivery_hours" "odh"
			ON "odh"."order_id" = "o"."id"
		WHERE "o"."delivery_group_id" IS NULL
			AND "o"."weight" <= ?
			AND "o"."regions" = ANY(?)
			AND "odh"."start_time" %s ?
			AND "odh"."end_time" >= ?
		ORDER BY "o"."weight" %s
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	orderingType := "DESC"
	if params.OrderByWeightASC {
		orderingType = "ASC"
	}

	startTimeOperator := "<="
	if params.WithGap {
		startTimeOperator = ">="
	}

	query = fmt.Sprintf(query, startTimeOperator, orderingType)

	err = db.Raw(
		query,
		params.MaxWeight,
		regionsArr,
		params.DeliveryHoursStart.Format("15:04:05"),
		params.DeliveryHoursEnd.Format("15:04:05"),
	).Scan(&order).Error
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, nil
	}

	var deliveryHours []OrderDeliveryHours
	err = db.Where("order_id = ?", order.ID).Find(&deliveryHours).Error
	if err != nil {
		return nil, err
	}
	order.DeliveryHours = deliveryHours

	res := toOrderEntity(*order)

	return &res, nil
}

func (s *OrderRepo) OrdersInGroup(groupID uint64) (*[]entity.Order, error) {
	orders := []Order{}

	err := s.gorm.Where(&Order{DeliveryGroupID: &groupID}).Preload("DeliveryHours").Find(&orders).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Order{}
	for _, o := range orders {
		res = append(res, toOrderEntity(o))
	}

	return &res, nil
}
