package bydate

import (
	"time"

	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/internal/repository/repositories"
)

type courierBatchState struct {
	DeliveryGroupRepo         *repositories.DeliveryGroupRepo
	courierID                 uint64
	CourierWorkingHoursID     uint64
	potential                 entity.DeliveryPotential
	isOnTheWay                bool
	currWeight                float64
	currOrders                uint
	currRegion                int32
	availableRegions          []int32
	courierType               entity.CourierType
	nextDeliveryDuration      time.Duration
	nextDeliveryStartDateTime time.Time
	shiftEndDateTime          time.Time
	deliveryGroup             *repositories.DeliveryGroup
}

func initCourierState(
	deliveryGroupRepo *repositories.DeliveryGroupRepo,
	courierID uint64,
	courierWorkingHoursID uint64,
	p entity.DeliveryPotential,
	t entity.CourierType,
	regions []int32,
	batchStart,
	batchEnd time.Time,
) (*courierBatchState, error) {
	duration, err := entity.NextDeliveryTimeInRegion(t, 0)
	if err != nil {
		return nil, err
	}

	return &courierBatchState{
		DeliveryGroupRepo:         deliveryGroupRepo,
		courierID:                 courierID,
		CourierWorkingHoursID:     courierWorkingHoursID,
		potential:                 p,
		isOnTheWay:                false,
		currWeight:                0,
		currOrders:                0,
		currRegion:                0,
		availableRegions:          regions,
		courierType:               t,
		nextDeliveryDuration:      duration,
		nextDeliveryStartDateTime: batchStart,
		shiftEndDateTime:          batchEnd,
		deliveryGroup:             nil,
	}, nil
}

func (c *courierBatchState) flush() error {

	duration, err := entity.NextDeliveryTimeInRegion(c.courierType, 0)
	if err != nil {
		return err
	}

	c.isOnTheWay = false
	c.currWeight = 0
	c.currOrders = 0
	c.currRegion = 0
	c.nextDeliveryDuration = duration

	if c.deliveryGroup != nil {

		err := c.DeliveryGroupRepo.Update(c.deliveryGroup)
		if err != nil {
			return err
		}

		c.deliveryGroup = nil
	}

	return nil
}

func (c *courierBatchState) addOrder(
	order entity.Order,
) (completeDateTime time.Time, discountCost uint32, err error) {

	dh := order.DeliveryHours[0]
	orderStartDateTime := time.Date(
		c.nextDeliveryStartDateTime.Year(),
		c.nextDeliveryStartDateTime.Month(),
		c.nextDeliveryStartDateTime.Day(),
		dh.StartTime.Hour(),
		dh.StartTime.Minute(),
		dh.StartTime.Second(),
		dh.StartTime.Nanosecond(),
		c.nextDeliveryStartDateTime.Location(),
	)

	if orderStartDateTime.After(c.nextDeliveryStartDateTime) {
		completeDateTime = orderStartDateTime.Add(c.nextDeliveryDuration)
	} else {
		// state on this point already flushed
		completeDateTime = c.nextDeliveryStartDateTime.Add(c.nextDeliveryDuration)
	}

	c.isOnTheWay = true
	c.currRegion = order.Regions
	c.currOrders++
	c.currWeight += order.Weight
	c.nextDeliveryStartDateTime = completeDateTime

	// calculate price with discount
	discount := entity.DeliveryInBatchCostDiscountPercents(c.currOrders)
	discountCost = order.Cost / 100 * (100 - discount)

	if c.deliveryGroup == nil {
		c.deliveryGroup, err = c.DeliveryGroupRepo.GetOrCreateGroup(
			c.courierID,
			c.CourierWorkingHoursID,
			c.shiftEndDateTime,
			completeDateTime.Add(-c.nextDeliveryDuration),
			completeDateTime,
		)
		if err != nil {
			return time.Time{}, 0, err
		}
	} else {
		c.deliveryGroup.EndDateTime = completeDateTime
	}

	duration, err := entity.NextDeliveryTimeInRegion(c.courierType, c.currOrders)
	if err != nil {
		return time.Time{}, 0, err
	}
	c.nextDeliveryDuration = duration

	return completeDateTime, discountCost, nil
}

func (c *courierBatchState) nextWillBeLast() bool {
	return c.potential.MaxOrders-1 == c.currOrders
}

func (c *courierBatchState) availableWeight() float64 {
	return c.potential.MaxWeight - c.currWeight
}

func (c *courierBatchState) isTimeToFlush() bool {
	if c.currOrders >= c.potential.MaxOrders || c.currWeight >= c.potential.MaxWeight {
		return true
	}

	return false
}

func (c *courierBatchState) isTimeToStop() bool {
	if c.shiftEndDateTime.Before(c.nextDeliveryStartDateTime.Add(c.nextDeliveryDuration)) {
		return true
	}

	return false
}
