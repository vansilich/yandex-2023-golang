package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"gopkg.in/go-playground/validator.v9"
	appErrors "yandex-team.ru/bstask/internal/errors"
)

func NewHttpServer() *echo.Echo {
	e := echo.New()

	e.Validator = &CustomValidator{Validator: validator.New()}
	e.HTTPErrorHandler = HttpErrorHandler

	return e
}

func HttpErrorHandler(err error, c echo.Context) {

	if c.Response().Committed {
		return
	}

	var appErr *appErrors.InternalError

	if errors.As(err, &appErr) {

		appErr = err.(*appErrors.InternalError)
		c.Logger().Error(appErr)

		if appErr.IsPublic {
			c.JSON(http.StatusBadRequest, appErr)
			return
		}
	}

	c.JSON(
		http.StatusBadRequest,
		http.StatusText(http.StatusInternalServerError),
	)
}
