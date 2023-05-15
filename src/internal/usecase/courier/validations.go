package courier

import (
	"reflect"

	"gopkg.in/go-playground/validator.v9"
	"yandex-team.ru/bstask/internal/entity"
)

func courier_type(fl validator.FieldLevel) bool {
	if fl.Field().Type().Kind() != reflect.String {
		return false
	}

	s, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	return entity.IsValidCourierType(s)
}
