package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask"
	"yandex-team.ru/bstask/config"
)

func NewHttpServer(conf config.AppConfig) *echo.Echo {
	e := echo.New()

	e.Validator = &CustomValidator{Validator: validator.New()}
	e.HTTPErrorHandler = HttpErrorHandler

	// setup middlewares
	if conf.Env != "test" {
		e.Use(middleware.RateLimiterWithConfig(RatelimiterConfig()))
	}

	return e
}

func HttpErrorHandler(err error, c echo.Context) {

	if c.Response().Committed {
		return
	}

	c.Logger().Error(err)

	var appErr *bstask.Error
	if errors.As(err, &appErr) {
		httpCode := bstask.ErrCodeToHTTPStatus(appErr)
		message := bstask.DefaultErrorMessage

		if httpCode < 500 {
			message = bstask.ErrorMessage(appErr)
		}

		c.JSON(httpCode, message)
		return
	}

	var echoError *echo.HTTPError
	if errors.As(err, &echoError) {
		c.JSON(echoError.Code, echoError.Message)
		return
	}

	c.JSON(
		http.StatusInternalServerError,
		http.StatusText(http.StatusInternalServerError),
	)
}
