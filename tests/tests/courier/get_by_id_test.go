package courier

import (
	"fmt"
	"net/http"
	"os"
	"tests/suites/postgres/fake"
	"tests/tests"

	"github.com/stretchr/testify/require"
)

var COURIER_GET_BY_ID_URL string = fmt.Sprintf("%s/couriers", os.Getenv("host"))

func (s *CourierTestSuite) TestGetByIdExpectValidationErrors() {

	requests := []string{
		fmt.Sprintf("%s/abc", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/_", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/0x0", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/0xF2", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/0x2", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/0", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/-1", COURIER_GET_BY_ID_URL),
		fmt.Sprintf("%s/92233720368547758070", COURIER_GET_BY_ID_URL),
	}

	for _, req := range requests {
		resp, err := http.Get(req)

		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "HTTP status code")
	}
}

func (s *CourierTestSuite) TestGetByIdErrorIfNotFound() {
	resp, err := http.Get(fmt.Sprintf("%s/%d", COURIER_GET_BY_ID_URL, 20))

	require.NoError(s.T(), err, "HTTP error")
	defer resp.Body.Close()

	require.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "HTTP status code")
}

func (s *CourierTestSuite) TestGetByIdExpectSuccess() {

	type SuccessResponse struct {
		CourierId    uint64   `json:"courier_id"`
		CourierType  string   `json:"courier_type"`
		Regions      []int32  `json:"regions"`
		WorkingHours []string `json:"working_hours"`
	}

	courier := fake.CorrectCouriers(1)[0]
	courierId := s.pgSuite.InsertCourier(courier)

	wh := fake.CorrectCourierWorkingHours(2)
	for _, i := range wh {
		i.CourierID = courierId
		s.pgSuite.InsertWorkingHours(i)
	}

	resp, err := http.Get(fmt.Sprintf("%s/%d", COURIER_GET_BY_ID_URL, courierId))
	require.NoError(s.T(), err, "HTTP error")
	defer resp.Body.Close()

	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "HTTP status code")

	var parsedRes SuccessResponse
	err = tests.ResponseToStruct(resp.Body, &parsedRes)
	require.NoError(s.T(), err, "Unmarshall")

	require.Equal(s.T(), courierId, parsedRes.CourierId)
	require.Equal(s.T(), courier.CourierType, parsedRes.CourierType)
	require.Equal(s.T(), courier.Regions, parsedRes.Regions)
	require.Equal(s.T(), tests.WorkingHoursToResponseFormat(wh), parsedRes.WorkingHours)
}
