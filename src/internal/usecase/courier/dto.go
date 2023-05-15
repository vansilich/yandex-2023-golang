package courier

import (
	"yandex-team.ru/bstask/internal/entity"
)

type CourierToCreateDTO struct {
	CourierType  string   `validate:"required,courier_type"`
	Regions      []int32  `validate:"required,unique"`
	WorkingHours []string `validate:"required,unique,each_HH_MM_HH_MM_time_interval"`
}

type CourierMetaDTO struct {
	Rating   *int32
	Earnings *int32
}

type AssignResponseGroupItem struct {
	CourierId uint64
	Orders    map[uint64]AssignOrdersGroup
}

type AssignOrdersGroup struct {
	GroupOrderId uint64
	Orders       []entity.Order
}
