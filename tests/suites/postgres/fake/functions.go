package fake

import (
	"tests/suites/postgres"
	"time"
)

func CorrectCouriers(count uint8) []postgres.Courier {
	res := make([]postgres.Courier, count, count)

	for i := 0; i < int(count); i++ {
		res[i] = postgres.Courier{
			Id:          correctId(),
			CourierType: correctCourierType(),
			Regions:     correctRegions(),
		}
	}

	return res
}

func CorrectCourierWorkingHours(count uint8) []postgres.CourierWorkingHours {
	res := make([]postgres.CourierWorkingHours, count, count)

	var end *time.Time = nil

	for i := 0; i < int(count); i++ {
		t := correctTime(end)
		start := t

		t = correctTime(&start)
		end = &t

		res[i] = postgres.CourierWorkingHours{
			ID:        correctId(),
			CourierID: correctId(),
			StartTime: start,
			EndTime:   *end,
		}
	}

	return res
}

func CorrectOrders(count uint8, asCompleted bool) []postgres.Order {
	res := make([]postgres.Order, count, count)

	for i := 0; i < int(count); i++ {
		item := postgres.Order{
			Id:      correctId(),
			Weight:  correctWeight(),
			Regions: correctRegions()[0],
			Cost:    correctCost(),
		}

		if asCompleted {
			delivId := correctId()
			completedAt := correctTime(nil)

			item.CompletedTime = &completedAt
			item.DeliveryGroupID = &delivId
		}

		res[i] = item
	}

	return res
}
