package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const TimeFormat = "15:04:05"

type Time time.Time

func NewTime(hour, min, sec int) Time {
	t := time.Date(0, time.January, 1, hour, min, sec, 0, time.UTC)
	return Time(t)
}

func (t *Time) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return t.UnmarshalText(string(v))
	case string:
		return t.UnmarshalText(v)
	case time.Time:
		*t = Time(v)
	case nil:
		*t = Time{}
	default:
		return fmt.Errorf("cannot sql.Scan() MyTime from: %#v", v)
	}

	return nil
}

func (t *Time) UnmarshalText(value string) error {
	dd, err := time.Parse(TimeFormat, value)
	if err != nil {
		return err
	}

	*t = Time(dd)

	return nil
}

func (t Time) Value() (driver.Value, error) {
	return driver.Value(time.Time(t).Format(TimeFormat)), nil
}

func (Time) GormDataType() string {
	return "TIME"
}
