package validations

import (
	"reflect"
	"regexp"

	"gopkg.in/go-playground/validator.v9"
)

func Each_HH_MM_time(fl validator.FieldLevel) bool {

	if fl.Field().Type().Kind() != reflect.Slice {
		return false
	}

	sl, ok := fl.Field().Interface().([]string)
	if !ok {
		return false
	}

	for _, item := range sl {
		match, err := regexp.Match(`^(0[0-9]|1[0-9]|2[0-3])\:(0[0-9]|[1-5][0-9])$`, []byte(item))
		if !match || err != nil {
			return false
		}
	}

	return true
}

func Each_HH_MM_HH_MM_time_interval(fl validator.FieldLevel) bool {

	if fl.Field().Type().Kind() != reflect.Slice {
		return false
	}

	sl, ok := fl.Field().Interface().([]string)
	if !ok {
		return false
	}

	for _, item := range sl {
		match, err := regexp.Match(`^(0[0-9]|1[0-9]|2[0-3])\:(0[0-9]|[1-5][0-9])-(0[0-9]|1[0-9]|2[0-3])\:(0[0-9]|[1-5][0-9])$`, []byte(item))
		if !match || err != nil {
			return false
		}
	}

	return true
}
