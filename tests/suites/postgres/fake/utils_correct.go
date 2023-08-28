package fake

import (
	"math/rand"
	"time"
)

var correctCourierTypes []string = []string{"FOOT", "BIKE", "AUTO"}

func correctId() uint64 {
	return rand.Uint64()
}

func correctCourierType() string {
	idx := rand.Intn(len(correctCourierTypes))
	return correctCourierTypes[idx]
}

func correctRegions() []int32 {
	count := rand.Intn(10)
	count++

	res := make([]int32, count)

	for i := 0; i < count; i++ {
		res = append(res, rand.Int31())
	}

	return res
}

func correctTime(after *time.Time) time.Time {

	if after == nil {
		hours := rand.Intn(23)
		minutes := rand.Intn(59)
		seconds := rand.Intn(59)

		return time.Date(0, 0, 0, hours, minutes, seconds, 0, time.UTC)
	}

	return time.Date(
		0,
		0,
		0,
		time.Time(*after).Hour()+rand.Intn(3),
		time.Time(*after).Minute()+rand.Intn(60),
		time.Time(*after).Second()+rand.Intn(60),
		0,
		time.UTC,
	)
}

func correctWeight() float64 {
	return rand.Float64()
}

func correctCost() uint32 {
	return rand.Uint32()
}
