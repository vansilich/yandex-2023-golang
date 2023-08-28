package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"yandex-team.ru/bstask/internal/http/controller"
)

type Router struct {
	Controllers Controllers
}

type Controllers struct {
	CourierController controller.CourierController
	OrderController   controller.OrderController
}

func NewRouter(cs Controllers) *Router {
	return &Router{
		Controllers: cs,
	}
}

func (r Router) SetupRoutes(e *echo.Echo) {

	e.GET("/ping", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "pong")
	})

	// courier methods
	e.GET("/couriers/assignments", r.Controllers.CourierController.Assignments)
	e.GET("/couriers", r.Controllers.CourierController.GetAll)
	e.POST("/couriers", r.Controllers.CourierController.Create)
	e.GET("/couriers/:courier_id", r.Controllers.CourierController.GetById)
	e.GET("/couriers/meta-info/:courier_id", r.Controllers.CourierController.MetaByCourierId)

	// order methods
	e.GET("/orders", r.Controllers.OrderController.GetAll)
	e.POST("/orders", r.Controllers.OrderController.Create)
	e.POST("/orders/complete", r.Controllers.OrderController.Complete)
	e.POST("/orders/assign", r.Controllers.OrderController.Assign)
	e.GET("/orders/:order_id", r.Controllers.OrderController.GetById)
}
