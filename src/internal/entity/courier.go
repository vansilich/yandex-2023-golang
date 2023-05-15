package entity

import (
	"time"

	appErrors "yandex-team.ru/bstask/internal/errors"
)

type Courier struct {
	ID           uint64
	CourierType  CourierType
	Regions      []int32
	WorkingHours []string
}

type DeliveryPotential struct {
	MaxWeight  float64
	MaxOrders  uint
	MaxRegions uint
}

type CourierType string

const (
	FOOT CourierType = "FOOT"
	BIKE CourierType = "BIKE"
	AUTO CourierType = "AUTO"
)

func ValidCourierTypes() []string {
	return []string{
		string(FOOT),
		string(BIKE),
		string(AUTO),
	}
}

func IsValidCourierType(t string) bool {
	validTypes := ValidCourierTypes()
	for _, validType := range validTypes {
		if validType == t {
			return true
		}
	}
	return false
}

func (c *Courier) SalaryRatio() (uint, error) {
	switch c.CourierType {
	case FOOT:
		return 2, nil
	case BIKE:
		return 3, nil
	case AUTO:
		return 4, nil
	default:
		return 0, appErrors.NewInternalError(nil, "invalid courier type", true)
	}
}

func (c *Courier) RatingRatio() (uint, error) {
	switch c.CourierType {
	case FOOT:
		return 3, nil
	case BIKE:
		return 2, nil
	case AUTO:
		return 1, nil
	default:
		return 0, appErrors.NewInternalError(nil, "invalid courier type", true)
	}
}

func DeliveryPotentialForType(t CourierType) (DeliveryPotential, error) {
	switch t {
	case FOOT:
		return DeliveryPotential{
			MaxWeight:  10,
			MaxOrders:  2,
			MaxRegions: 1,
		}, nil
	case BIKE:
		return DeliveryPotential{
			MaxWeight:  20,
			MaxOrders:  4,
			MaxRegions: 2,
		}, nil
	case AUTO:
		return DeliveryPotential{
			MaxWeight:  40,
			MaxOrders:  7,
			MaxRegions: 3,
		}, nil
	default:
		return DeliveryPotential{}, appErrors.NewInternalError(nil, "invalid courier type", true)
	}
}

func DeliveryInBatchCostDiscountPercents(ordersCountInBatch uint) uint32 {
	if ordersCountInBatch <= 1 {
		return 0
	}

	return 20
}

func NextDeliveryTimeInRegion(t CourierType, ordersCountInBatch uint) (time.Duration, error) {
	switch t {
	case FOOT:
		if ordersCountInBatch > 0 {
			return time.ParseDuration("10m")
		} else {
			return time.ParseDuration("25m")
		}
	case BIKE:
		if ordersCountInBatch > 0 {
			return time.ParseDuration("8m")
		} else {
			return time.ParseDuration("12m")
		}
	case AUTO:
		if ordersCountInBatch > 0 {
			return time.ParseDuration("4m")
		} else {
			return time.ParseDuration("8m")
		}
	default:
		return 0, appErrors.NewInternalError(nil, "invalid courier type", true)
	}
}
