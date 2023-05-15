package repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"yandex-team.ru/bstask/internal/entity"
	appErrors "yandex-team.ru/bstask/internal/errors"
	"yandex-team.ru/bstask/pkg/gorm/types"
)

// @migration
type Courier struct {
	ID           uint64 `gorm:"primaryKey"`
	CourierType  string
	Regions      pq.Int32Array         `gorm:"type:integer[]"`
	WorkingHours []CourierWorkingHours `gorm:"foreignKey:CourierID;references:ID"`
}

// @migration
type CourierWorkingHours struct {
	ID        uint64 `gorm:"primaryKey"`
	CourierID uint64
	Courier   *Courier `gorm:"foreignKey:CourierID"`
	StartTime types.Time
	EndTime   types.Time
}

type CourierRepo struct {
	gorm *gorm.DB
}

var (
	CourierNotFoundError = appErrors.NewInternalError(nil, "Courier not found", true)
)

func NewCourierRepo(grm *gorm.DB) *CourierRepo {
	return &CourierRepo{
		gorm: grm,
	}
}

func (s *CourierRepo) Atomic(fn func(repo CourierRepo) error) error {
	tx := s.gorm.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	newRepo := CourierRepo{gorm: tx}
	err := fn(newRepo)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

type CourierToCreateDTO struct {
	CourierType  string
	Regions      []int32
	WorkingHours []CourierWorkingHoursIntervalDTO
}

type CourierWorkingHoursIntervalDTO struct {
	StartTime time.Time
	EndTime   time.Time
}

func (s *CourierRepo) BatchCreate(newCouriers []CourierToCreateDTO) (*[]entity.Courier, error) {

	couriers := []Courier{}

	err := s.Atomic(func(repo CourierRepo) error {
		for _, c := range newCouriers {

			// BUG: if we assign `working hours` to courier here,
			// we will deal with duplicate INSERT queries

			couriers = append(couriers, Courier{
				CourierType: c.CourierType,
				Regions:     c.Regions,
			})
		}

		dbRes := repo.gorm.CreateInBatches(couriers, 20)
		if dbRes.Error != nil {
			return dbRes.Error
		}

		for i, c := range newCouriers {
			courier := &couriers[i]

			for _, wh := range c.WorkingHours {
				(*courier).WorkingHours = append(courier.WorkingHours, CourierWorkingHours{
					Courier:   courier,
					StartTime: types.NewTime(wh.StartTime.Hour(), wh.StartTime.Minute(), wh.StartTime.Second()),
					EndTime:   types.NewTime(wh.EndTime.Hour(), wh.EndTime.Minute(), wh.EndTime.Second()),
				})
			}

			dbRes = repo.gorm.Save(courier)
			if dbRes.Error != nil {
				return dbRes.Error
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	res := []entity.Courier{}
	for _, c := range couriers {

		wh := []string{}
		for _, t := range c.WorkingHours {
			st := time.Time(t.StartTime).Format("15:04")
			et := time.Time(t.EndTime).Format("15:04")

			wh = append(wh, st+"-"+et)
		}

		res = append(res, entity.Courier{
			ID:           c.ID,
			CourierType:  entity.CourierType(c.CourierType),
			Regions:      c.Regions,
			WorkingHours: wh,
		})
	}

	return &res, nil
}

func (s *CourierRepo) FindById(id uint64) (*entity.Courier, error) {

	var courier Courier

	err := s.gorm.Model(&Courier{}).Preload("WorkingHours").Find(&courier, int(id)).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {

			CourierNotFoundError.Err = err
			return nil, CourierNotFoundError
		}

		return nil, err
	}

	wh := []string{}
	for _, t := range courier.WorkingHours {
		st := time.Time(t.StartTime).Format("15:04")
		et := time.Time(t.EndTime).Format("15:04")

		wh = append(wh, st+"-"+et)
	}

	return &entity.Courier{
		ID:           courier.ID,
		CourierType:  entity.CourierType(courier.CourierType),
		Regions:      courier.Regions,
		WorkingHours: wh,
	}, nil
}

func (s *CourierRepo) PaginatedFetchAll(offset, limit int32) (*[]entity.Courier, error) {

	couriers := []Courier{}

	err := s.gorm.Model(&Courier{}).Preload("WorkingHours").Limit(int(limit)).Offset(int(offset)).Find(&couriers).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Courier{}
	for _, c := range couriers {

		wh := []string{}
		for _, t := range c.WorkingHours {
			st := time.Time(t.StartTime).Format("15:04")
			et := time.Time(t.EndTime).Format("15:04")

			wh = append(wh, st+"-"+et)
		}

		res = append(res, entity.Courier{
			ID:           c.ID,
			CourierType:  entity.CourierType(c.CourierType),
			Regions:      c.Regions,
			WorkingHours: wh,
		})
	}

	return &res, nil
}

func (s *CourierRepo) WorkingIntervalForDelivery(courierID uint64, start, end time.Time) (*CourierWorkingHours, error) {
	var wh *CourierWorkingHours

	err := s.gorm.Where(
		"start_time <= ? AND end_time >= ? AND courier_id = ?",
		start.Format("15:04:05"),
		end.Format("15:04:05"),
		courierID,
	).First(&wh).Error

	if err != nil {
		return nil, err
	}

	return wh, nil
}

type AllWorkingHoursRes struct {
	CourierID      uint64
	CourierType    string
	Regions        []int32
	WorkingHoursID uint64
	StartTime      time.Time
	EndTime        time.Time
}

func (s *CourierRepo) AllWorkingHoursByCourierType(courierType entity.CourierType) (*[]AllWorkingHoursRes, error) {

	tmp := []struct {
		CourierID      uint64        `gorm:"column:courier_id"`
		CourierType    string        `gorm:"column:courier_type"`
		Regions        pq.Int32Array `gorm:"column:regions"`
		WorkingHoursID uint64        `gorm:"column:working_hours_id"`
		StartTime      types.Time    `gorm:"column:start_time"`
		EndTime        types.Time    `gorm:"column:end_time"`
	}{}

	err := s.gorm.Raw(`
		SELECT 
			"c"."id" as "courier_id",
			"c"."courier_type" as "courier_type",
			"c"."regions" as "regions",
			"cwh"."id" as "working_hours_id",
			"cwh"."start_time" as "start_time",
			"cwh"."end_time" as "end_time"
		FROM "couriers" as "c"
		LEFT JOIN "courier_working_hours" as cwh 
			ON "cwh"."courier_id" = "c"."id"
		WHERE "c"."courier_type" = ?
		ORDER BY "cwh"."start_time" ASC`,
		string(courierType),
	).Scan(&tmp).Error

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	res := []AllWorkingHoursRes{}
	for _, d := range tmp {
		res = append(res, AllWorkingHoursRes{
			CourierID:      d.CourierID,
			CourierType:    d.CourierType,
			Regions:        d.Regions,
			WorkingHoursID: d.WorkingHoursID,
			StartTime:      time.Time(d.StartTime),
			EndTime:        time.Time(d.EndTime),
		})
	}

	return &res, nil
}
