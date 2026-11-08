package controller

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	outSuccess    = -1
	outExists     = 0
	outNotFound   = 1
	outNotAllowed = 3
)

type Server struct {
	auth.UnimplementedAuthServiceServer
	pool           *pgxpool.Pool
	allowBootstrap bool
}

func NewServer(pool *pgxpool.Pool, allowBootstrap bool) *Server {
	return &Server{pool: pool, allowBootstrap: allowBootstrap}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Server) requireAdmin(ctx context.Context, userID int32) error {
	var role, status int16
	err := s.pool.QueryRow(ctx, `select role, status from account.creds where id = $1`, userID).Scan(&role, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAdminRequired
	}
	if err != nil {
		return internal(ctx, "checking admin", err)
	}
	if role != int16(auth.Role_ROLE_ADMIN) || status != int16(auth.UserStatus_USER_STATUS_ACTIVE) {
		return ErrAdminRequired
	}
	return nil
}

func (s *Server) callOutcode(ctx context.Context, action, query string, args ...any) (int16, error) {
	var outcode int16
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&outcode); err != nil {
		return 0, internal(ctx, action, err)
	}
	return outcode, nil
}

func internal(ctx context.Context, action string, err error) error {
	slog.LogAttrs(ctx, slog.LevelError, action, slog.Any("error", err))
	return ErrInternal
}
