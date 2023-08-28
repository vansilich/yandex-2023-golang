package postgres

import "time"

type Courier struct {
	Id          uint64  `db:"id"`
	CourierType string  `db:"courier_type"`
	Regions     []int32 `db:"regions"`
}

type CourierWorkingHours struct {
	ID        uint64    `db:"id"`
	CourierID uint64    `db:"courier_id"`
	StartTime time.Time `db:"start_time"`
	EndTime   time.Time `db:"end_time"`
}

type DeliveryGroup struct {
	ID                    uint64    `db:"id"`
	CourierID             uint64    `db:"courier_id"`
	CourierWorkingHoursID uint64    `db:"courier_working_hours_id"`
	AssignDate            time.Time `db:"assign_date"`
	StartDateTime         time.Time `db:"start_date_time"`
	EndDateTime           time.Time `db:"end_date_time"`
}

type Order struct {
	Id              uint64     `db:"id"`
	Weight          float64    `db:"weight"`
	Regions         int32      `db:"regions"`
	Cost            uint32     `db:"cost"`
	CompletedTime   *time.Time `db:"completed_time"`
	DeliveryGroupID *uint64    `db:"delivery_group_id"`
}
