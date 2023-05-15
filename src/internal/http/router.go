package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	e.GET("/couriers/assignments", r.Controllers.CourierController.Assignments, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.GET("/couriers", r.Controllers.CourierController.GetAll, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.POST("/couriers", r.Controllers.CourierController.Create, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.GET("/couriers/:courier_id", r.Controllers.CourierController.GetById, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.GET("/couriers/meta-info/:courier_id", r.Controllers.CourierController.MetaByCourierId, middleware.RateLimiterWithConfig(RatelimiterConfig()))

	// order methods
	e.GET("/orders", r.Controllers.OrderController.GetAll, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.POST("/orders", r.Controllers.OrderController.Create, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.POST("/orders/complete", r.Controllers.OrderController.Complete, middleware.RateLimiterWithConfig(RatelimiterConfig()))
	e.POST("/orders/assign", r.Controllers.OrderController.Assign)
	e.GET("/orders/:order_id", r.Controllers.OrderController.GetById, middleware.RateLimiterWithConfig(RatelimiterConfig()))
}

func RatelimiterConfig() middleware.RateLimiterConfig {
	return middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 10, Burst: 0, ExpiresIn: time.Second},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}
}
