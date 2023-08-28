package courier

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"tests/suites/postgres"
	"tests/suites/postgres/fake"
	"tests/tests"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/stretchr/testify/require"
)

var COURIER_GET_ALL_URL string = fmt.Sprintf("%s/couriers", os.Getenv("host"))

func (s *CourierTestSuite) TestGetAllExpectValidationErrors() {

	requests := []string{
		fmt.Sprintf("%s?limit=abc", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?offset=abc", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?limit=123&offset=abc", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?limit=abc&offset=1123", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?limit=-121", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?offset=-143", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?offset=0x143", COURIER_GET_ALL_URL),
		fmt.Sprintf("%s?limit=0xAF", COURIER_GET_ALL_URL),
	}

	for _, req := range requests {
		resp, err := http.Get(req)

		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "HTTP status code")
	}
}

func (s *CourierTestSuite) TestGetAllExpectSuccess() {

	type SuccessResponseItem struct {
		CourierId    uint64   `json:"courier_id"`
		CourierType  string   `json:"courier_type"`
		Regions      []int32  `json:"regions"`
		WorkingHours []string `json:"working_hours"`
	}
	type SuccessResponse struct {
		Couriers []SuccessResponseItem `json:"couriers"`
		Offset   int32                 `json:"offset"`
		Limit    int32                 `json:"limit"`
	}

	couriers := fake.CorrectCouriers(15)

	for _, c := range couriers {
		courierId := s.pgSuite.InsertCourier(c)

		wh := fake.CorrectCourierWorkingHours(3)
		for _, i := range wh {
			i.CourierID = courierId
			s.pgSuite.InsertWorkingHours(i)
		}
	}

	requests := []struct {
		reqUrl string
		limit  int32
		offset int32
	}{
		{
			reqUrl: COURIER_GET_ALL_URL,
			limit:  1,
			offset: 0,
		},
		{
			reqUrl: fmt.Sprintf("%s?limit=%d", COURIER_GET_ALL_URL, 10),
			limit:  10,
			offset: 0,
		},
		{
			reqUrl: fmt.Sprintf("%s?offset=%d", COURIER_GET_ALL_URL, 5),
			limit:  1,
			offset: 5,
		},
		{
			reqUrl: fmt.Sprintf("%s?limit=%d&offset=%d", COURIER_GET_ALL_URL, 5, 5),
			limit:  5,
			offset: 5,
		},
		{
			reqUrl: fmt.Sprintf("%s?limit=%d&offset=%d", COURIER_GET_ALL_URL, 5, 15),
			limit:  5,
			offset: 15,
		},
		{
			reqUrl: fmt.Sprintf("%s?limit=%d&offset=%d", COURIER_GET_ALL_URL, 0, 10),
			limit:  0,
			offset: 10,
		},
		{
			reqUrl: fmt.Sprintf("%s?limit=%d&offset=%d", COURIER_GET_ALL_URL, 20, 0),
			limit:  20,
			offset: 0,
		},
	}

	for _, req := range requests {
		resp, err := http.Get(req.reqUrl)

		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusOK, resp.StatusCode, "HTTP status code")

		var parsedRes SuccessResponse
		err = tests.ResponseToStruct(resp.Body, &parsedRes)
		require.NoError(s.T(), err, "Unmarshall")

		rows, err := s.pgSuite.Pgx.Query(
			context.Background(),
			fmt.Sprintf(
				`SELECT * FROM couriers ORDER BY id ASC OFFSET %d LIMIT %d`,
				req.offset,
				req.limit,
			),
		)
		require.NoError(s.T(), err)

		expectedRes := SuccessResponse{
			Couriers: []SuccessResponseItem{},
			Limit:    req.limit,
			Offset:   req.offset,
		}

		var dbCouriers []postgres.Courier
		require.NoError(s.T(), pgxscan.ScanAll(&dbCouriers, rows), "Scan w.h. DB model")

		for _, c := range dbCouriers {
			rows, err = s.pgSuite.Pgx.Query(
				s.pgSuite.Ctx,
				"SELECT * FROM courier_working_hours WHERE courier_id = $1",
				c.Id,
			)

			var dbWorkingHours []postgres.CourierWorkingHours
			require.NoError(s.T(), pgxscan.ScanAll(&dbWorkingHours, rows), "Scan w.h. DB model")

			expectedRes.Couriers = append(expectedRes.Couriers, SuccessResponseItem{
				CourierId:    c.Id,
				CourierType:  c.CourierType,
				Regions:      c.Regions,
				WorkingHours: tests.WorkingHoursToResponseFormat(dbWorkingHours),
			})
		}

		require.Equal(s.T(), expectedRes, parsedRes)
	}
}
