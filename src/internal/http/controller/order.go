package controller

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"yandex-team.ru/bstask/internal/usecase/order"
)

type OrderController struct {
	uc *order.OrderUseCase
}

type OrderDto struct {
	ID            uint64     `json:"order_id"`
	Weight        float64    `json:"weight"`
	Regions       int32      `json:"regions"`
	DeliveryHours []string   `json:"delivery_hours"`
	Cost          uint32     `json:"cost"`
	CompletedTime *time.Time `json:"completed_time"`
}

func NewOrderController(uc *order.OrderUseCase) OrderController {
	return OrderController{
		uc: uc,
	}
}

// ===================================
// ========== GET /orders ============
// ===================================
func (c *OrderController) GetAll(ctx echo.Context) error {

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

	orders, err := c.uc.PaginatedGetAll(int32(offset), int32(limit))
	if err != nil {
		return err
	}

	res := []OrderDto{}

	for _, order := range *orders {

		dh := []string{}
		for _, t := range order.DeliveryHours {
			dh = append(dh, t.StartTime.Format("15:04")+"-"+t.EndTime.Format("15:04"))
		}

		res = append(res, OrderDto{
			ID:            order.ID,
			Weight:        order.Weight,
			Regions:       order.Regions,
			DeliveryHours: dh,
			Cost:          order.Cost,
			CompletedTime: order.CompletedTime,
		})
	}

	return ctx.JSON(200, res)
}

// ===================================

// ====================================
// ========== POST /orders ============
// ====================================
type OrderCreateRequest struct {
	Orders []OrderRequestCreateDto `json:"orders" validate:"required,dive"`
}

type OrderRequestCreateDto struct {
	Weight        float64  `json:"weight" validate:"required,min=0"`
	Regions       int32    `json:"regions" validate:"required,max=2147483647"`
	DeliveryHours []string `json:"delivery_hours" validate:"required"`
	Cost          uint32   `json:"cost" validate:"required,min=0,max=2147483647"`
}

type OrderCreateResponse struct {
	Orders []OrderDto `json:"orders"`
}

func (c *OrderController) Create(ctx echo.Context) error {

	var req OrderCreateRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := ctx.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	newOrders := []order.OrderToCreateDTO{}
	for _, newCourier := range req.Orders {

		newOrders = append(newOrders, order.OrderToCreateDTO{
			Weight:        newCourier.Weight,
			Regions:       newCourier.Regions,
			DeliveryHours: newCourier.DeliveryHours,
			Cost:          newCourier.Cost,
		})
	}

	savedOrders, err := c.uc.CreateOrders(newOrders)
	if err != nil {
		return err
	}

	res := OrderCreateResponse{}
	res.Orders = []OrderDto{}

	for _, newOrder := range *savedOrders {

		dh := []string{}
		for _, t := range newOrder.DeliveryHours {
			dh = append(dh, t.StartTime.Format("15:04")+"-"+t.EndTime.Format("15:04"))
		}

		res.Orders = append(res.Orders, OrderDto{
			ID:            newOrder.ID,
			Weight:        newOrder.Weight,
			Regions:       newOrder.Regions,
			DeliveryHours: dh,
			Cost:          newOrder.Cost,
		})
	}

	return ctx.JSON(200, res)
}

// ====================================

// ==============================================
// ========== POST /orders/:order_id ============
// ==============================================

func (c *OrderController) GetById(ctx echo.Context) error {

	orderIdParam := ctx.Param("order_id")

	orderId, err := strconv.Atoi(orderIdParam)
	if err != nil || orderId <= 0 || orderId > math.MaxInt64 {
		return echo.NewHTTPError(http.StatusBadRequest, ":order_id must be valid int64")
	}

	order, err := c.uc.GetById(uint64(orderId))
	if err != nil {
		return err
	}

	dh := []string{}
	for _, t := range order.DeliveryHours {
		dh = append(dh, t.StartTime.Format("15:04")+"-"+t.EndTime.Format("15:04"))
	}

	return ctx.JSON(200, OrderDto{
		ID:            order.ID,
		Weight:        order.Weight,
		Regions:       order.Regions,
		DeliveryHours: dh,
		Cost:          order.Cost,
		CompletedTime: order.CompletedTime,
	})
}

// =============================================

// =============================================
// ========== POST /orders/complete ============
// =============================================

type OrderCompleteRequest struct {
	Info []OrderCompleteItem `json:"complete_info" validate:"required,dive"`
}

type OrderCompleteItem struct {
	CourierId    int64     `json:"courier_id" validate:"min=0,max=9223372036854775807"`
	OrderId      int64     `json:"order_id" validate:"min=0,max=9223372036854775807"`
	CompleteTime time.Time `json:"complete_time" validate:"required"`
}

func (c *OrderController) Complete(ctx echo.Context) error {

	var req OrderCompleteRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := ctx.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	toComplete := []order.OrderToCompleteDTO{}
	for _, i := range req.Info {
		toComplete = append(toComplete, order.OrderToCompleteDTO{
			CourierId:    i.CourierId,
			OrderId:      i.OrderId,
			CompleteTime: i.CompleteTime,
		})
	}

	orders, err := c.uc.Complete(toComplete)
	if err != nil {
		return err
	}

	res := []OrderDto{}
	if orders != nil {
		for _, o := range *orders {

			dh := []string{}
			for _, t := range o.DeliveryHours {
				dh = append(dh, t.StartTime.Format("15:04")+"-"+t.EndTime.Format("15:04"))
			}

			res = append(res, OrderDto{
				ID:            o.ID,
				Weight:        o.Weight,
				Regions:       o.Regions,
				DeliveryHours: dh,
				Cost:          o.Cost,
				CompletedTime: o.CompletedTime,
			})
		}
	}

	return ctx.JSON(200, res)
}

// =============================================

// ===========================================
// ========== POST /orders/assign ============
// ===========================================

type OrderAssignByDateResponseItem struct {
	Date     string                    `json:"date"`
	Couriers []AssignResponseGroupItem `json:"couriers"`
}

type AssignResponseGroupItem struct {
	CourierId uint64              `json:"courier_id"`
	Orders    []AssignOrdersGroup `json:"orders"`
}

type AssignOrdersGroup struct {
	GroupOrderId uint64     `json:"group_order_id"`
	Orders       []OrderDto `json:"orders"`
}

func (c *OrderController) Assign(ctx echo.Context) error {

	assignDate := time.Now()

	dateParam := ctx.QueryParam("date")
	if dateParam != "" {

		var err error
		assignDate, err = time.Parse("2006-01-02", dateParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Bad date format")
		}
	}

	assigns, err := c.uc.AssignByDate(assignDate)
	if err != nil {
		return err
	}

	res := OrderAssignByDateResponseItem{
		Date: assignDate.Format("2006-01-02"),
	}

	for _, courier := range assigns.Couriers {

		assignResponseGroupItem := AssignResponseGroupItem{
			CourierId: courier.CourierId,
		}
		for _, orderGroup := range courier.Orders {

			assignOrdersGroup := AssignOrdersGroup{
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

	return ctx.JSON(200, []OrderAssignByDateResponseItem{res})
}
