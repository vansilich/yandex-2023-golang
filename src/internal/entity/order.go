package entity

import "time"

type Order struct {
	ID              uint64
	Weight          float64
	Regions         int32
	DeliveryHours   []OrderDeliveryHours
	Cost            uint32
	CompletedTime   *time.Time
	DeliveryGroupID *uint64
}

type OrderDeliveryHours struct {
	ID        uint64
	StartTime time.Time
	EndTime   time.Time
}
