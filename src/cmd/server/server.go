package main

import (
	trmgorm "github.com/avito-tech/go-transaction-manager/gorm"
	"github.com/avito-tech/go-transaction-manager/trm/manager"
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

	appConf := config.NewAppConfig()

	// db.AutoMigrate(
	// 	&repositories.Courier{},
	// 	&repositories.CourierWorkingHours{},
	// 	&repositories.Order{},
	// 	&repositories.OrderDeliveryHours{},
	// 	&repositories.DeliveryGroup{},
	// )

	courierRepo := repositories.NewCourierRepo(db, trmgorm.DefaultCtxGetter)
	orderRepo := repositories.NewOrderRepo(db, trmgorm.DefaultCtxGetter)
	deliveryGroupRepo := repositories.NewOrderGroupRepo(db, trmgorm.DefaultCtxGetter)

	m, err := manager.New(trmgorm.NewDefaultFactory(db))
	if err != nil {
		panic(err)
	}

	courierUseCase := courier.New(m, courierRepo, orderRepo, deliveryGroupRepo)
	orderUseCase := order.New(m, orderRepo, courierRepo, deliveryGroupRepo)

	cs := http.Controllers{
		CourierController: controller.NewCourierController(courierUseCase),
		OrderController:   controller.NewOrderController(orderUseCase),
	}
	r := http.NewRouter(cs)

	e := http.NewHttpServer(appConf)
	r.SetupRoutes(e)

	e.Logger.Fatal(e.Start(":8080"))
}
