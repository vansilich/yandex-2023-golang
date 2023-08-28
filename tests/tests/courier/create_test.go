package courier

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"tests/suites/postgres"
	"tests/tests"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var COURIER_CREATE_URL string = fmt.Sprintf("%s/couriers", os.Getenv("host"))

func (s *CourierTestSuite) TestCreateExpectValidationErrors() {

	requests := []string{
		`{
			"couriers": [
				{
					"courier_type": "invalid_type",
					"regions": [123],
					"working_hours": ["10:00-12:00"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": 123,
					"working_hours": ["10:00-12:00"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [123],
					"working_hours": ["150:00-12:00"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [123],
					"working_hours": ["12:00"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [123],
					"working_hours": ["11:00-12:60"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [123],
					"working_hours": []
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [],
					"working_hours": ["11:00-12:60"]
				}
			]
		}`,
		`{
			"couriers": [
				{
					"courier_type": "FOOT",
					"regions": [123, 321],
					"working_hours": []
				}
			]
		}`,
	}

	for _, req := range requests {
		resp, err := http.Post(COURIER_CREATE_URL, "application/json", strings.NewReader(req))

		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "HTTP status code")
	}

	var cnt int
	err := s.pgSuite.Pgx.QueryRow(s.pgSuite.Ctx, "SELECT COUNT(*) FROM couriers").Scan(&cnt)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0, cnt)
}

func (s *CourierTestSuite) TestCreateExpectSuccess() {
	type SuccessResponseItem struct {
		CourierId    uint64   `json:"courier_id"`
		CourierType  string   `json:"courier_type"`
		Regions      []int32  `json:"regions"`
		WorkingHours []string `json:"working_hours"`
	}
	type SuccessResponse struct {
		Couriers []SuccessResponseItem `json:"couriers"`
	}

	requests := []struct {
		req string
		res SuccessResponse
	}{
		{
			req: `{
				"couriers": [
					{
						"courier_type": "FOOT",
						"regions": [123, 213, 23213],
						"working_hours": ["10:00-12:00"]
					}
				]
			}`,
			res: SuccessResponse{
				Couriers: []SuccessResponseItem{
					{
						CourierId:    1,
						CourierType:  "FOOT",
						Regions:      []int32{123, 213, 23213},
						WorkingHours: []string{"10:00-12:00"},
					},
				},
			},
		},
		{
			req: `{
				"couriers": [
					{
						"courier_type": "BIKE",
						"regions": [123, 930293, 1293],
						"working_hours": ["15:03-21:58"]
					}
				]
			}`,
			res: SuccessResponse{
				Couriers: []SuccessResponseItem{
					{
						CourierId:    2,
						CourierType:  "BIKE",
						Regions:      []int32{123, 930293, 1293},
						WorkingHours: []string{"15:03-21:58"},
					},
				},
			},
		},
		{
			req: `{
				"couriers": [
					{
						"courier_type": "AUTO",
						"regions": [123, 930293, 1293],
						"working_hours": ["23:00-01:35"]
					},
					{
						"courier_type": "FOOT",
						"regions": [930293, 129300000],
						"working_hours": ["23:00-01:35", "02:00-03:41"]
					}
				]
			}`,
			res: SuccessResponse{
				Couriers: []SuccessResponseItem{
					{
						CourierId:    3,
						CourierType:  "AUTO",
						Regions:      []int32{123, 930293, 1293},
						WorkingHours: []string{"23:00-01:35"},
					},
					{
						CourierId:    4,
						CourierType:  "FOOT",
						Regions:      []int32{930293, 129300000},
						WorkingHours: []string{"23:00-01:35", "02:00-03:41"},
					},
				},
			},
		},
	}

	countAll := 0
	for _, item := range requests {
		resp, err := http.Post(COURIER_CREATE_URL, "application/json", strings.NewReader(item.req))
		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusOK, resp.StatusCode, "HTTP status code")

		var parsedRes SuccessResponse
		err = tests.ResponseToStruct(resp.Body, &parsedRes)
		require.NoError(s.T(), err, "Unmarshall")

		require.Equal(s.T(), item.res, parsedRes)

		countAll += len(item.res.Couriers)

		for _, courier := range item.res.Couriers {
			rows, err := s.pgSuite.Pgx.Query(
				s.pgSuite.Ctx,
				"SELECT * FROM couriers WHERE id = $1",
				courier.CourierId,
			)
			require.NoError(s.T(), err, "Get courier from DB")

			rows.Next()

			var dbCourier postgres.Courier
			require.NoError(s.T(), pgxscan.ScanRow(&dbCourier, rows), "Scan courier DB model")

			rows.Close()

			require.Equal(s.T(), courier.CourierId, dbCourier.Id)
			require.Equal(s.T(), courier.CourierType, dbCourier.CourierType)
			require.EqualValues(s.T(), courier.Regions, dbCourier.Regions)

			rows, err = s.pgSuite.Pgx.Query(
				s.pgSuite.Ctx,
				"SELECT * FROM courier_working_hours WHERE courier_id = $1",
				courier.CourierId,
			)
			require.NoError(s.T(), err, "Get courier w.h. from DB")

			var dbWorkingHours []postgres.CourierWorkingHours
			require.NoError(s.T(), pgxscan.ScanAll(&dbWorkingHours, rows), "Scan w.h. DB model")

			wh := tests.WorkingHoursToResponseFormat(dbWorkingHours)

			assert.Subset(s.T(), courier.WorkingHours, wh)
			assert.Len(s.T(), wh, len(courier.WorkingHours))
		}
	}

	var actualCountAll int
	err := s.pgSuite.Pgx.QueryRow(s.pgSuite.Ctx, "SELECT COUNT(*) FROM couriers").Scan(&actualCountAll)
	require.NoError(s.T(), err)

	require.Equal(s.T(), countAll, actualCountAll)
}
