package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"yandex-team.ru/bstask"
	"yandex-team.ru/bstask/internal/entity"
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
	gorm      *gorm.DB
	ctxGetter *trmgorm.CtxGetter
}

func NewCourierRepo(grm *gorm.DB, c *trmgorm.CtxGetter) *CourierRepo {
	return &CourierRepo{
		gorm:      grm,
		ctxGetter: c,
	}
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

func toCourierEntity(c Courier) entity.Courier {

	wh := []string{}
	for _, t := range c.WorkingHours {
		st := time.Time(t.StartTime).Format("15:04")
		et := time.Time(t.EndTime).Format("15:04")

		wh = append(wh, st+"-"+et)
	}

	return entity.Courier{
		ID:           c.ID,
		CourierType:  entity.CourierType(c.CourierType),
		Regions:      c.Regions,
		WorkingHours: wh,
	}
}

func (s *CourierRepo) BatchCreate(ctx context.Context, newCouriers []CourierToCreateDTO) (*[]entity.Courier, error) {

	couriers := []Courier{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)

	for _, c := range newCouriers {

		// BUG: if we assign `working hours` to courier here,
		// we will deal with duplicate INSERT queries

		couriers = append(couriers, Courier{
			CourierType: c.CourierType,
			Regions:     c.Regions,
		})
	}

	dbRes := db.CreateInBatches(couriers, 20)
	if dbRes.Error != nil {
		return nil, dbRes.Error
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

		dbRes = db.Save(courier)
		if dbRes.Error != nil {
			return nil, dbRes.Error
		}
	}

	res := []entity.Courier{}
	for _, c := range couriers {
		res = append(res, toCourierEntity(c))
	}

	return &res, nil
}

func (s *CourierRepo) FindById(ctx context.Context, id uint64) (*entity.Courier, error) {

	var courier Courier

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Model(&Courier{}).Preload("WorkingHours").Where("id = ?", id).First(&courier).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &bstask.Error{
				Op:      "repositories.CourierRepo.FindById",
				Code:    bstask.ENOTFOUND,
				Err:     err,
				Message: "courier not found",
				Fields: map[string]interface{}{
					"courier_id": id,
				},
			}
		}

		return nil, err
	}

	entity := toCourierEntity(courier)

	return &entity, nil
}

func (s *CourierRepo) PaginatedFetchAll(ctx context.Context, offset, limit int32) (*[]entity.Courier, error) {

	couriers := []Courier{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Model(&Courier{}).Preload("WorkingHours").Limit(int(limit)).Offset(int(offset)).Find(&couriers).Error
	if err != nil {
		return nil, err
	}

	res := []entity.Courier{}
	for _, c := range couriers {
		res = append(res, toCourierEntity(c))
	}

	return &res, nil
}

func (s *CourierRepo) WorkingIntervalForDelivery(ctx context.Context, courierID uint64, start, end time.Time) (*CourierWorkingHours, error) {
	var wh *CourierWorkingHours

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Where(
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

func (s *CourierRepo) AllWorkingHoursByCourierType(ctx context.Context, courierType entity.CourierType) (*[]AllWorkingHoursRes, error) {

	tmp := []struct {
		CourierID      uint64        `gorm:"column:courier_id"`
		CourierType    string        `gorm:"column:courier_type"`
		Regions        pq.Int32Array `gorm:"column:regions"`
		WorkingHoursID uint64        `gorm:"column:working_hours_id"`
		StartTime      types.Time    `gorm:"column:start_time"`
		EndTime        types.Time    `gorm:"column:end_time"`
	}{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Raw(`
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
