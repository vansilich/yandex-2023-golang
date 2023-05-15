package entity

import "time"

type DeliveryGroup struct {
	ID                    uint64
	CourierID             uint64
	CourierWorkingHoursID uint64
	AssignDate            time.Time
	StartDateTime         time.Time
	EndDateTime           time.Time
}
