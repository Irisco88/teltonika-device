package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/openfms/avldb/migrations/golang"
	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"
)

//go:embed migrations/*
var embedMigrations embed.FS

const (
	SQLDirPath    = "migrations/sqls"
	GolangDirPath = "migrations/golang"
	DefaultDriver = "pgx"
)

var (
	database      string
	driver        string
	migrationPath string
)
var usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-by-one            Migrate the DB up by 1
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
`

func main() {
	app := &cli.App{
		Name:      "migration",
		Usage:     "db migration management",
		UsageText: "--driver pgx --database 'postgres://admin:12345678@127.0.0.1:7505/testdb?sslmode=disable' --path 'migrations/sqls' up",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "database",
				Usage:       "database url or dsn",
				Required:    true,
				Aliases:     []string{"url", "dsn"},
				EnvVars:     []string{"DATABASE_URL"},
				Destination: &database,
			},
			&cli.StringFlag{
				Name:        "path",
				Value:       SQLDirPath,
				DefaultText: fmt.Sprintf("[%s | %s]", SQLDirPath, GolangDirPath),
				Usage:       "migrations path",
				Aliases:     []string{"p"},
				Required:    true,
				EnvVars:     []string{"MIGRATIONS_PATH"},
				Action: func(context *cli.Context, s string) error {
					if !slices.Contains([]string{SQLDirPath, GolangDirPath}, s) {
						return fmt.Errorf("mifration path must be in [%s,%s]", SQLDirPath, GolangDirPath)
					}
					return nil
				},
				Destination: &migrationPath,
			},
			&cli.StringFlag{
				Name:        "driver",
				Usage:       "database driver",
				Value:       DefaultDriver,
				DefaultText: DefaultDriver,
				Aliases:     []string{"d"},
				EnvVars:     []string{"DATABASE_DRIVER"},
				Destination: &driver,
			},
		},
		ArgsUsage: usageCommands,
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()
			if len(args) == 0 {
				return fmt.Errorf("command not found")
			}
			command := ctx.Args().Get(0)
			goose.SetBaseFS(embedMigrations)
			db, err := goose.OpenDBWithDriver(driver, database)
			if err != nil {
				return fmt.Errorf("migration: failed to open DB: %v\n", err)
			}

			defer func() {
				if err := db.Close(); err != nil {
					log.Fatalf("migration: failed to close DB: %v\n", err)
				}
			}()
			if err := goose.Run(command, db, migrationPath, args...); err != nil {
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
