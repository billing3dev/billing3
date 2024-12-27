package database

import (
	"context"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"os"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema
var sqls embed.FS

var Q *Queries
var Conn *pgxpool.Pool

func Init() error {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE"))
	if err != nil {
		return fmt.Errorf("pgx parse config: %w", err)
	}

	config.AfterConnect = func(ctx context.Context, p *pgx.Conn) error {
		pgxdecimal.Register(p.TypeMap())
		return nil
	}

	Conn, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("pgx connect: %w", err)
	}

	Q = New(Conn)

	// create tables

	bytes, err := sqls.ReadFile("schema/000-schema.sql")
	if err != nil {
		slog.Error("create tables", "err", err)
		panic(err)
	}

	_, err = Conn.Exec(context.Background(), string(bytes))
	if err != nil {
		slog.Error("create tables", "err", err)
		panic(err)
	}

	return nil
}

func Close() {
	Conn.Close()
}
