package config

type PgsqlConnectionConf struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

type DatabaseConfig struct {
	Pgsql PgsqlConnectionConf
}

func DatabaseConf() *DatabaseConfig {
	return &DatabaseConfig{
		Pgsql: PgsqlConnectionConf{
			Host:     "db",
			Port:     5432,
			Database: "postgres",
			Username: "postgres",
			Password: "password",
		},
	}
}
