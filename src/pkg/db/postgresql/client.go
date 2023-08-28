package postgresql

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const CONNECTION_RETRIES = 5
const CONNECTION_RETRY_BACKOF = 5

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

		logger := logger.Default.LogMode(logger.Warn)
		retries := CONNECTION_RETRIES
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})

		for err != nil {
			retries--
			if retries < 0 {
				log.Fatalf("Could not connect to database: %v", err)
			}

			time.Sleep(CONNECTION_RETRY_BACKOF * time.Second)

			logger.Warn(
				context.TODO(),
				fmt.Sprintf("Failed to connect to postgres. Doing retry %d of %d", retries, CONNECTION_RETRIES),
			)

			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})
		}

		instance = db
	})

	return instance
}
