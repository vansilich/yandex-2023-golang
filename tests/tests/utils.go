package tests

import (
	"encoding/json"
	"io"
	"tests/suites/postgres"
	"time"
)

func WorkingHoursToResponseFormat(wh []postgres.CourierWorkingHours) []string {
	res := []string{}

	for _, interval := range wh {
		start := time.Time(interval.StartTime).Format("15:04")
		end := time.Time(interval.EndTime).Format("15:04")

		res = append(res, start+"-"+end)
	}

	return res
}

func ResponseToStruct[T interface{}](responseBody io.ReadCloser, toStruct *T) error {

	body, err := io.ReadAll(responseBody)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, toStruct)
	if err != nil {
		return err
	}

	return nil
}
