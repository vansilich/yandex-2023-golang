package courier

import (
	"fmt"
	"net/http"
	"os"
	"tests/suites/postgres/fake"
	"tests/tests"

	"github.com/stretchr/testify/require"
)

type CourierMetaByIDResponse struct {
	CourierId    uint64   `json:"courier_id"`
	CourierType  string   `json:"courier_type"`
	Regions      []int32  `json:"regions"`
	WorkingHours []string `json:"working_hours"`
	Rating       *int32   `json:"rating,omitempty"`
	Earnings     *int32   `json:"earnings,omitempty"`
}

var COURIER_META_BY_ID_URL string = fmt.Sprintf("%s/couriers/meta-info", os.Getenv("host"))

func (s *CourierTestSuite) TestMetaByIdExpectValidationErrors() {

	courier := fake.CorrectCouriers(1)[0]
	courierId := s.pgSuite.InsertCourier(courier)

	wh := fake.CorrectCourierWorkingHours(2)
	for _, i := range wh {
		i.CourierID = courierId
		s.pgSuite.InsertWorkingHours(i)
	}

	requests := []string{
		fmt.Sprintf("%s/abc", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/_", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/0x0", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/0xF2", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/0x2", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/0", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/-1", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/92233720368547758070", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?startDate=2023-01-01", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?endDate=2023-01-01", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?startDate=2023-01-01&endDate=2023-01", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?startDate=2023-13-01&endDate=2024-01-01", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/%d?startDate=2024-12-01&endDate=2024-01-01", COURIER_META_BY_ID_URL, courierId),
		fmt.Sprintf("%s/8?startDate=2023-12-32&endDate=2024-01-01", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?startDate=2023-12-01&endDate=2024-01-32", COURIER_META_BY_ID_URL),
		fmt.Sprintf("%s/8?startDate=2023-02-29&endDate=2023-03-01", COURIER_META_BY_ID_URL),
	}

	for _, req := range requests {
		resp, err := http.Get(req)

		require.NoError(s.T(), err, "HTTP error")
		defer resp.Body.Close()

		require.Equal(s.T(), http.StatusBadRequest, resp.StatusCode, "HTTP status code")
	}
}

func (s *CourierTestSuite) TestMetaByIdErrorIfCourierNotFound() {

	resp, err := http.Get(fmt.Sprintf(
		"%s/%d?startDate=%s&endDate=%s",
		COURIER_META_BY_ID_URL,
		20,
		"2023-01-01",
		"2023-02-01",
	))

	require.NoError(s.T(), err, "HTTP error")
	defer resp.Body.Close()

	require.Equal(s.T(), http.StatusNotFound, resp.StatusCode, "HTTP status code")
}

func (s *CourierTestSuite) TestMetaByIdRatingAndEarningsEmptyIfNoOrdersFound() {

	courier := fake.CorrectCouriers(1)[0]
	courierId := s.pgSuite.InsertCourier(courier)

	wh := fake.CorrectCourierWorkingHours(2)
	for _, i := range wh {
		i.CourierID = courierId
		s.pgSuite.InsertWorkingHours(i)
	}

	os.Exit(1)

	resp, err := http.Get(fmt.Sprintf(
		"%s/%d?startDate=%s&endDate=%s",
		COURIER_META_BY_ID_URL,
		courierId,
		"2023-01-01",
		"2023-02-01",
	))

	require.NoError(s.T(), err, "HTTP error")
	defer resp.Body.Close()

	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "HTTP status code")

	var parsedRes CourierMetaByIDResponse
	err = tests.ResponseToStruct(resp.Body, &parsedRes)
	require.NoError(s.T(), err, "Unmarshall")

	require.Equal(s.T(), courierId, parsedRes.CourierId)
	require.Equal(s.T(), courier.CourierType, parsedRes.CourierType)
	require.Equal(s.T(), courier.Regions, parsedRes.Regions)
	require.Equal(s.T(), tests.WorkingHoursToResponseFormat(wh), parsedRes.WorkingHours)
	require.Nil(s.T(), parsedRes.Earnings)
	require.Nil(s.T(), parsedRes.Rating)
}

func (s *CourierTestSuite) TestMetaByIdExpectSuccess() {

	couriers := fake.CorrectCouriers(5)
	for _, courier := range couriers {
		courierId := s.pgSuite.InsertCourier(courier)
		wh := fake.CorrectCourierWorkingHours(2)
		for _, i := range wh {
			i.CourierID = courierId
			s.pgSuite.InsertWorkingHours(i)
		}
	}

	// orders := fake.CorrectOrders(22, false)
	// for _, order := range orders {
	// 	orderId := s.pgSuite.InsertCourier
	// }
}
