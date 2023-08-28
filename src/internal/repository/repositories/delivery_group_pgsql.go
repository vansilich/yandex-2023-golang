package repositories

import (
	"context"
	"time"

	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"gorm.io/gorm"
	"yandex-team.ru/bstask/internal/entity"
	"yandex-team.ru/bstask/pkg/gorm/types"
)

// @migration
type DeliveryGroup struct {
	ID                    uint64               `gorm:"primaryKey"`
	CourierID             uint64               `gorm:"not null"`
	Courier               *Courier             `gorm:"foreignKey:CourierID"`
	CourierWorkingHoursID uint64               `gorm:"not null"`
	WorkingHours          *CourierWorkingHours `gorm:"foreignKey:CourierWorkingHoursID"`
	AssignDate            types.Date           `gorm:"not null"`
	StartDateTime         time.Time            `gorm:"not null"`
	EndDateTime           time.Time            `gorm:"not null"`
}

type DeliveryGroupRepo struct {
	gorm      *gorm.DB
	ctxGetter *trmgorm.CtxGetter
}

func NewOrderGroupRepo(grm *gorm.DB, c *trmgorm.CtxGetter) *DeliveryGroupRepo {
	return &DeliveryGroupRepo{
		gorm:      grm,
		ctxGetter: c,
	}
}

func (s *DeliveryGroupRepo) CreateGroup(
	ctx context.Context,
	courierID uint64,
	workingHoursID uint64,
	Date time.Time,
	startDateTime time.Time,
	endDateTime time.Time,
) (*DeliveryGroup, error) {

	var res DeliveryGroup

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Where(DeliveryGroup{
		CourierID:             courierID,
		CourierWorkingHoursID: workingHoursID,
		AssignDate:            types.Date(Date),
		StartDateTime:         startDateTime,
		EndDateTime:           endDateTime,
	}).FirstOrCreate(&res).Error

	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *DeliveryGroupRepo) Update(ctx context.Context, group *DeliveryGroup) error {

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Save(group).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *DeliveryGroupRepo) AllByDate(ctx context.Context, date time.Time) (*[]entity.DeliveryGroup, error) {

	groups := []DeliveryGroup{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Where(
		"assign_date >= ? AND assign_date < ?",
		date.Format("2006-01-02"),
		date.AddDate(0, 0, 1).Format("2006-01-02"),
	).Find(&groups).Error
	if err != nil {
		return nil, err
	}

	res := []entity.DeliveryGroup{}
	for _, g := range groups {
		res = append(res, entity.DeliveryGroup{
			ID:                    g.ID,
			CourierID:             g.CourierID,
			CourierWorkingHoursID: g.CourierWorkingHoursID,
			AssignDate:            time.Time(g.AssignDate),
			StartDateTime:         g.StartDateTime,
			EndDateTime:           g.EndDateTime,
		})
	}

	return &res, nil
}

func (s *DeliveryGroupRepo) AllByDateAndIds(ctx context.Context, courierIDs []uint64, date time.Time) (*[]entity.DeliveryGroup, error) {

	groups := []DeliveryGroup{}

	db := s.ctxGetter.DefaultTrOrDB(ctx, s.gorm).WithContext(ctx)
	err := db.Where(
		"assign_date >= ? AND assign_date < ? AND courier_id IN ?",
		date.Format("2006-01-02"),
		date.AddDate(0, 0, 1).Format("2006-01-02"),
		courierIDs).Find(&groups).Error
	if err != nil {
		return nil, err
	}

	res := []entity.DeliveryGroup{}
	for _, g := range groups {
		res = append(res, entity.DeliveryGroup{
			ID:                    g.ID,
			CourierID:             g.CourierID,
			CourierWorkingHoursID: g.CourierWorkingHoursID,
			AssignDate:            time.Time(g.AssignDate),
			StartDateTime:         g.StartDateTime,
			EndDateTime:           g.EndDateTime,
		})
	}

	return &res, nil
}

// func (s *DeliveryGroupRepo) ByCourierIdInInterval(ctx context.Context courier) (*entity.DeliveryGroup, error) {

// }
