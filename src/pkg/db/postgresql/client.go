package postgresql

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var onceDb sync.Once
var instance *gorm.DB

func GetInstance(host, user, password, dbname string, port int) *gorm.DB {

	onceDb.Do(func() {

		dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s",
			host,
			port,
			user,
			dbname,
			password,
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			log.Fatalf("Could not connect to database :%v", err)
		}

		instance = db
	})

	return instance
}
