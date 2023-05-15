package controller

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"yandex-team.ru/bstask/internal/usecase/courier"
)

type CourierController struct {
	uc *courier.CourierUseCase
}

type CourierDto struct {
	CourierId    uint64   `json:"courier_id"`
	CourierType  string   `json:"courier_type"`
	Regions      []int32  `json:"regions"`
	WorkingHours []string `json:"working_hours"`
}

func NewCourierController(uc *courier.CourierUseCase) CourierController {
	return CourierController{
		uc: uc,
	}
}

// ===============================================
// ========== GET /couriers/assignments ==========
// ===============================================

type CourierAssignmentsResponseItem struct {
	Date     string                        `json:"date"`
	Couriers []CourierAssignmentsGroupItem `json:"couriers"`
}

type CourierAssignmentsGroupItem struct {
	CourierId uint64                          `json:"courier_id"`
	Orders    []CourierAssignmentsOrdersGroup `json:"orders"`
}

type CourierAssignmentsOrdersGroup struct {
	GroupOrderId uint64     `json:"group_order_id"`
	Orders       []OrderDto `json:"orders"`
}

func (c *CourierController) Assignments(ctx echo.Context) error {

	date := time.Now()

	dateParam := ctx.QueryParam("date")
	if dateParam != "" {

		var err error
		date, err = time.Parse("2006-01-02", dateParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Bad date format")
		}
	}

	courierIDs := []uint64{}
	courierIdParam := ctx.QueryParam("courier_id")
	if courierIdParam != "" {

		id, err := strconv.Atoi(courierIdParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Bad courier_id format")
		}

		courierIDs = append(courierIDs, uint64(id))
	}

	assignments, err := c.uc.Assignments(courierIDs, date)
	if err != nil {
		return err
	}

	res := CourierAssignmentsResponseItem{
		Date: date.Format("2006-01-02"),
	}

	for _, courier := range assignments {

		assignResponseGroupItem := CourierAssignmentsGroupItem{
			CourierId: courier.CourierId,
		}
		for _, orderGroup := range courier.Orders {

			assignOrdersGroup := CourierAssignmentsOrdersGroup{
				GroupOrderId: orderGroup.GroupOrderId,
			}
			for _, order := range orderGroup.Orders {

				dh := []string{}
				for _, t := range order.DeliveryHours {
					dh = append(dh, t.StartTime.Format("15:04")+"-"+t.EndTime.Format("15:04"))
				}

				assignOrdersGroup.Orders = append(assignOrdersGroup.Orders, OrderDto{
					ID:            order.ID,
					Weight:        order.Weight,
					Regions:       order.Regions,
					DeliveryHours: dh,
					Cost:          order.Cost,
					CompletedTime: order.CompletedTime,
				})
			}

			assignResponseGroupItem.Orders = append(assignResponseGroupItem.Orders, assignOrdersGroup)
		}

		res.Couriers = append(res.Couriers, assignResponseGroupItem)
	}

	return ctx.JSON(200, []CourierAssignmentsResponseItem{res})
}

// ===============================================

// ===================================
// ========== GET /couriers ==========
// ===================================

type CourierGetAllReponse struct {
	Couriers []CourierDto `json:"couriers"`
	Offset   int32        `json:"offset"`
	Limit    int32        `json:"limit"`
}

func (c *CourierController) GetAll(ctx echo.Context) error {

	var limit int = 1
	var offset int = 0
	var err error

	limitParam := ctx.QueryParam("limit")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit < 0 || limit > math.MaxInt32 {
			return echo.NewHTTPError(400, "Invalid 'limit' param")
		}
	}

	offsetParam := ctx.QueryParam("offset")
	if offsetParam != "" {
		offset, err = strconv.Atoi(offsetParam)
		if err != nil || offset < 0 || offset > math.MaxInt32 {
			return echo.NewHTTPError(400, "Invalid 'offset' param")
		}
	}

	couriers, err := c.uc.PaginatedGetAll(int32(offset), int32(limit))
	if err != nil {
		return err
	}

	res := CourierGetAllReponse{
		Couriers: []CourierDto{},
	}
	for _, courier := range *couriers {
		res.Couriers = append(res.Couriers, CourierDto{
			CourierId:    courier.ID,
			CourierType:  string(courier.CourierType),
			Regions:      courier.Regions,
			WorkingHours: courier.WorkingHours,
		})
	}
	res.Offset = int32(offset)
	res.Limit = int32(limit)

	return ctx.JSON(200, res)
}

// ====================================
// ========== POST /couriers ==========
// ====================================
type CourierCreateRequest struct {
	Couriers []CourierRequestCreateDto `json:"couriers" validate:"required,dive"`
}

type CourierRequestCreateDto struct {
	CourierType  string   `json:"courier_type" validate:"required"`
	Regions      []int32  `json:"regions" validate:"required"`
	WorkingHours []string `json:"working_hours" validate:"required"`
}

type CourierCreateResponse struct {
	Couriers []CourierDto `json:"couriers" validate:"required"`
}

func (c *CourierController) Create(ctx echo.Context) error {

	var req CourierCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := ctx.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	newCouriers := []courier.CourierToCreateDTO{}
	for _, newCourier := range req.Couriers {

		newCouriers = append(newCouriers, courier.CourierToCreateDTO{
			CourierType:  newCourier.CourierType,
			Regions:      newCourier.Regions,
			WorkingHours: newCourier.WorkingHours,
		})
	}

	savedCouriers, err := c.uc.CreateCouriers(newCouriers)
	if err != nil {
		return err
	}

	res := CourierCreateResponse{}

	for _, newCourier := range *savedCouriers {
		res.Couriers = append(res.Couriers, CourierDto{
			CourierId:    newCourier.ID,
			CourierType:  string(newCourier.CourierType),
			Regions:      newCourier.Regions,
			WorkingHours: newCourier.WorkingHours,
		})
	}

	return ctx.JSON(200, res)
}

// ====================================

// ================================================
// ========== GET /couriers/{courier_id} ==========
// ================================================

func (c *CourierController) GetById(ctx echo.Context) error {

	courierIdParam := ctx.Param("courier_id")

	courierId, err := strconv.Atoi(courierIdParam)
	if err != nil || courierId <= 0 || courierId > math.MaxInt64 {
		return echo.NewHTTPError(http.StatusBadRequest, ":courier_id must be valid integer")
	}

	courier, err := c.uc.GetById(uint64(courierId))
	if err != nil {
		return err
	}

	return ctx.JSON(200, CourierDto{
		CourierId:    courier.ID,
		CourierType:  string(courier.CourierType),
		Regions:      courier.Regions,
		WorkingHours: courier.WorkingHours,
	})
}

// ================================================

// ==========================================================
// ========== GET /couriers/meta-info/{courier_id} ==========
// ==========================================================

type CourierMetaByIDResponse struct {
	CourierDto
	Rating   *int32 `json:"rating,omitempty"`
	Earnings *int32 `json:"earnings,omitempty"`
}

func (c *CourierController) MetaByCourierId(ctx echo.Context) error {

	courierId, err := strconv.Atoi(ctx.Param("courier_id"))
	if err != nil || courierId <= 0 || courierId > math.MaxInt64 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid :courier_id param")
	}

	startDate, err := time.Parse("2006-01-02", ctx.QueryParam("startDate"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid :startDate param")
	}

	endDate, err := time.Parse("2006-01-02", ctx.QueryParam("endDate"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid :endDate param")
	}

	courier, err := c.uc.GetById(uint64(courierId))
	if err != nil {
		return err
	}

	meta, err := c.uc.MetaInInterval(courier, startDate, endDate)
	if err != nil {
		return err
	}

	res := CourierMetaByIDResponse{
		CourierDto: CourierDto{
			CourierId:    courier.ID,
			CourierType:  string(courier.CourierType),
			Regions:      courier.Regions,
			WorkingHours: courier.WorkingHours,
		},
	}

	if meta.Rating != nil {
		res.Rating = meta.Rating
	}

	if meta.Earnings != nil {
		res.Earnings = meta.Earnings
	}

	return ctx.JSON(http.StatusOK, res)
}
