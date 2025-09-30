package db

import (
	"context"
	"fmt"
	"time"

	"guangfu250923/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(cfg config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSL)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Println("Error parsing database configuration:", err)
		return nil, err
	}
	poolCfg.MaxConns = 5
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return pgxpool.NewWithConfig(ctx, poolCfg)
}
