package main

import (
	"yandex-team.ru/bstask/config"
	"yandex-team.ru/bstask/internal/http"
	"yandex-team.ru/bstask/internal/http/controller"
	"yandex-team.ru/bstask/internal/repository/repositories"
	"yandex-team.ru/bstask/internal/usecase/courier"
	"yandex-team.ru/bstask/internal/usecase/order"
	"yandex-team.ru/bstask/pkg/db/postgresql"
)

// TODO поработать с созданием и проверкой delivery_groups
// TODO проверить, что количество регионов соответствую типу курьера

func main() {

	dbConf := config.DatabaseConf()
	db := postgresql.GetInstance(
		dbConf.Pgsql.Host,
		dbConf.Pgsql.Username,
		dbConf.Pgsql.Password,
		dbConf.Pgsql.Database,
		dbConf.Pgsql.Port,
	)

	db.AutoMigrate(
		&repositories.Courier{},
		&repositories.CourierWorkingHours{},
		&repositories.Order{},
		&repositories.OrderDeliveryHours{},
		&repositories.DeliveryGroup{},
	)

	courierRepo := repositories.NewCourierRepo(db)
	orderRepo := repositories.NewOrderRepo(db)
	deliveryGroupRepo := repositories.NewOrderGroupRepo(db)

	courierUseCase := courier.New(courierRepo, orderRepo, deliveryGroupRepo)
	orderUseCase := order.New(orderRepo, courierRepo, deliveryGroupRepo)

	cs := http.Controllers{
		CourierController: controller.NewCourierController(courierUseCase),
		OrderController:   controller.NewOrderController(orderUseCase),
	}
	r := http.NewRouter(cs)

	e := http.NewHttpServer()
	r.SetupRoutes(e)

	e.Logger.Fatal(e.Start(":8080"))
}
