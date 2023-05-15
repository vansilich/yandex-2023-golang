package order

import "time"

type OrderToCreateDTO struct {
	Weight        float64  `validate:"required"`
	Regions       int32    `validate:"required"`
	DeliveryHours []string `validate:"required,unique,each_HH_MM_HH_MM_time_interval"`
	Cost          uint32   `validate:"required"`
}

type OrderToCompleteDTO struct {
	CourierId    int64     `json:"courier_id" validate:"min=0,max=9223372036854775807"`
	OrderId      int64     `json:"order_id" validate:"min=0,max=9223372036854775807"`
	CompleteTime time.Time `json:"complete_time" validate:"required"`
}
