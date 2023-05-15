package bydate

import (
	"time"

	"yandex-team.ru/bstask/internal/entity"
)

type AssignResponseGroup struct {
	Date     time.Time
	Couriers []AssignResponseGroupItem
}

type AssignResponseGroupItem struct {
	CourierId uint64
	Orders    map[uint64]AssignOrdersGroup
}

type AssignOrdersGroup struct {
	GroupOrderId uint64
	Orders       []entity.Order
}
