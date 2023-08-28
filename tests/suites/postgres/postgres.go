package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	migratePgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const MIGRATIONS_TABLE = "migrations"

type Suite struct {
	Ctx     context.Context
	Pgx     *pgx.Conn
	migrate *migrate.Migrate
}

func SetupInstance(ctx context.Context) *Suite {

	dbHost := os.Getenv("POSTRGES_HOST")
	dbPort, err := strconv.Atoi(os.Getenv("POSTRGES_PORT"))
	if err != nil {
		panic("unable to recognise POSTRGES_PORT env variable")
	}
	dbUser := os.Getenv("POSTRGES_USER")
	dbPassword := os.Getenv("POSTRGES_PASSWORD")
	dbName := os.Getenv("POSTRGES_DB_NAME")

	suite := Suite{
		Ctx: ctx,
	}

	pgConnInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)
	db, err := sql.Open("pgx", pgConnInfo)
	if err != nil {
		panic(fmt.Errorf("connection to postgres failed: %w", err))
	}

	if err = db.Ping(); err != nil {
		panic(fmt.Errorf("DB ping error: %w", err))
	}

	_, filename, _, _ := runtime.Caller(0)
	migrationsPath := "file://" + path.Join(path.Dir(filename), "migrations")

	driver, err := migratePgx.WithInstance(db, &migratePgx.Config{
		MigrationsTable: MIGRATIONS_TABLE,
		DatabaseName:    dbName,
	})
	if err != nil {
		panic(fmt.Errorf("failed to create migration driver: %w", err))
	}

	suite.migrate, err = migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		panic(fmt.Errorf("failed to create migration instance: %w", err))
	}

	suite.migrate.Down()

	if err = suite.migrate.Up(); err != nil {
		panic(fmt.Errorf("failed to up migrations: %w", err))
	}

	pgxConnUrl := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName,
	)
	suite.Pgx, err = pgx.Connect(ctx, pgxConnUrl)
	if err != nil {
		panic(fmt.Errorf("failed to connect by pgx: %w", err))
	}

	return &suite
}

func (suite *Suite) TearDownInstance() {
	err := suite.migrate.Down()
	if err != nil {
		fmt.Println("paniced here")
		panic(fmt.Errorf("failed to rollback db migrations: %w", err))
	}

	err = suite.Pgx.Close(suite.Ctx)
	if err != nil {
		panic(fmt.Errorf("failed close sql.DB connection: %w", err))
	}
}

func (suite *Suite) TruncateAll() {
	dbUser := os.Getenv("POSTRGES_USER")
	query := `
		SELECT tablename FROM pg_tables
		WHERE tableowner = $1 AND schemaname = 'public'`

	rows, err := suite.Pgx.Query(suite.Ctx, query, dbUser)
	if err != nil {
		panic(fmt.Errorf("failed to get list of tables: %w", err))
	}

	tableNames := []string{}
	for rows.Next() {
		var tableName string
		if err = rows.Scan(&tableName); err != nil {
			panic(fmt.Errorf("failed to scan table name: %w", err))
		}

		if tableName == MIGRATIONS_TABLE {
			continue
		}

		tableNames = append(tableNames, tableName)
	}
	rows.Close()

	for _, tableName := range tableNames {
		query := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)
		if _, err := suite.Pgx.Exec(suite.Ctx, query); err != nil {
			panic(fmt.Errorf("failed to truncate table: %w", err))
		}
	}
}
