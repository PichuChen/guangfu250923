package handlers

import (
	"guangfu250923/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
	s3   *storage.S3Uploader
}

func New(pool *pgxpool.Pool, s3 *storage.S3Uploader) *Handler { return &Handler{pool: pool, s3: s3} }
