package routes

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// e.GET("/ping", ping)
func ping(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "pong")
}