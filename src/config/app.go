package config

import "os"

type AppConfig struct {
	Env string // test, dev or prod
}

func NewAppConfig() AppConfig {

	conf := AppConfig{
		Env: os.Getenv("APP_ENV"),
	}

	return conf
}
