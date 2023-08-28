package postgres

import (
	"context"
	"fmt"
)

func (s *Suite) InsertCourier(courier Courier) uint64 {

	var id uint64
	err := s.Pgx.QueryRow(
		context.Background(),
		`INSERT INTO couriers (courier_type, regions) VALUES ($1, $2) RETURNING "id"`,
		courier.CourierType,
		courier.Regions,
	).Scan(&id)

	if err != nil {
		panic(fmt.Errorf("error in insert: %w", err))
	}

	return id
}

func (s *Suite) InsertWorkingHours(wh CourierWorkingHours) uint64 {

	var id uint64
	query := `INSERT INTO courier_working_hours 
		(courier_id, start_time, end_time)
		VALUES 
		($1, $2, $3) 
	RETURNING "id"`

	err := s.Pgx.QueryRow(
		context.Background(), query, wh.CourierID, wh.StartTime, wh.EndTime,
	).Scan(&id)

	if err != nil {
		panic(fmt.Errorf("error in insert: %w", err))
	}

	return id
}

func (s Suite) InsertDeliveryGroup(dg DeliveryGroup) uint64 {
	var id uint64
	query := `INSERT INTO delivery_groups 
		(courier_id, courier_working_hours_id, assign_date, start_date_time, end_date_time)
		VALUES 
		($1, $2, $3, $4, $5) 
	RETURNING "id"`

	err := s.Pgx.QueryRow(
		context.Background(),
		query,
		dg.CourierID,
		dg.CourierWorkingHoursID,
		dg.AssignDate,
		dg.StartDateTime,
		dg.EndDateTime,
	).Scan(&id)

	if err != nil {
		panic(fmt.Errorf("error in insert: %w", err))
	}

	return id
}

func (s *Suite) InsertOrder(order Order) uint64 {

	var id uint64
	query := `INSERT INTO orders 
		(weight, regions, cost, completed_time, delivery_group_id)
		VALUES 
		($1, $2, $3, $4, $5) 
	RETURNING "id"`

	err := s.Pgx.QueryRow(
		context.Background(),
		query,
		order.Weight,
		order.Regions,
		order.Cost,
		order.CompletedTime,
		order.DeliveryGroupID,
	).Scan(&id)

	if err != nil {
		panic(fmt.Errorf("error in insert: %w", err))
	}

	return id
}
