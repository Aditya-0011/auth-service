package controller

import (
	"auth/db"
	"auth/internal/password"
	"auth/internal/timeout"
	"context"
	"log/slog"
	"time"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	server struct {
		auth.UnimplementedAuthServiceServer
		postgres *db.PostgresParams
	}

	ServerParams struct {
		Postgres *db.PostgresParams
	}
)

func NewServer(params ServerParams) *server {
	return &server{
		postgres: params.Postgres,
	}
}

func (s *server) Login(c context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	ctx, cancel := timeout.WithDeadline(c, timeout.Duration)
	defer cancel()

	var (
		count      int
		id         *int
		dbPassword *[]byte
	)

	query := `
		with total as (select count(*) as cnt from account.creds)
		select total.cnt, c.id, c.password 
		from total 
		left join account.creds c on c.email = $1
	`
	err := s.postgres.Pool.QueryRow(ctx, query, req.GetEmail()).Scan(&count, &id, &dbPassword)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "error querying database",
			slog.String("error", err.Error()))
		return nil, ErrInternal
	}

	if count == 0 {
		hashedPassword, err := password.Hash([]byte(req.GetPassword()))
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "error generating hash",
				slog.String("error", err.Error()))
			return nil, ErrInternal
		}

		var (
			newId   int
			outcode int16
		)

		err = s.postgres.Pool.QueryRow(ctx, "select * from account.create($1, $2)", req.GetEmail(), hashedPassword).Scan(&newId, &outcode)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "error creating user",
				slog.String("error", err.Error()))
			return nil, ErrInternalCreate
		}

		switch outcode {
		case -1:
			return &auth.LoginResponse{UserId: int32(newId), UserEmail: req.GetEmail()}, nil
		case 0: // currently only supports single user - Aditya Punmiya
			return nil, ErrUserNotFound
		case 1:
			return nil, ErrUserNotFound
		default:
			return nil, ErrInternalCreate
		}

	} else {
		if id == nil || dbPassword == nil {
			return nil, ErrUnauthenticated
		}

		match, err := password.Compare([]byte(req.GetPassword()), *dbPassword)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "error verifying password",
				slog.String("error", err.Error()))
			return nil, ErrInternal
		}

		if !match {
			return nil, ErrUnauthenticated
		}

		return &auth.LoginResponse{UserId: int32(*id), UserEmail: req.GetEmail()}, nil
	}
}

func (s *server) GetKey(c context.Context, req *auth.KeyRequest) (*auth.GetKeyResponse, error) {
	ctx, cancel := timeout.WithDeadline(c, timeout.Duration)
	defer cancel()

	var (
		key       string
		updatedAt time.Time
	)

	query := `select key, updatedAt from account.creds where id = $1`

	err := s.postgres.Pool.QueryRow(ctx, query, req.GetUserId()).Scan(&key, &updatedAt)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "error querying database",
			slog.String("error", err.Error()))
		return nil, ErrInternal
	}

	if key == "" {
		return nil, ErrKeyNotFound
	}

	return &auth.GetKeyResponse{
		Key:       key,
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

func (s *server) RotateKey(c context.Context, req *auth.KeyRequest) (*auth.SimpleResponse, error) {
	ctx, cancel := timeout.WithDeadline(c, timeout.Duration)
	defer cancel()

	var outcode int16

	err := s.postgres.Pool.QueryRow(ctx, `select outcode from account.rotate_key($1)`, req.GetUserId()).Scan(&outcode)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "error rotating key",
			slog.Int("userId", int(req.GetUserId())),
			slog.String("error", err.Error()))
		return nil, ErrInternal
	}

	switch outcode {
	case -1:
		return &auth.SimpleResponse{Message: "Key rotated successfully"}, nil
	case 1:
		return nil, ErrUserNotFound
	default:
		return nil, ErrInternal
	}
}

func (s *server) ValidateKey(c context.Context, req *auth.ValidateKeyRequest) (*auth.LoginResponse, error) {
	ctx, cancel := timeout.WithDeadline(c, timeout.Duration)
	defer cancel()

	var (
		userId *int
		email  *string
	)

	query := `select id, email from account.creds where key = $1`
	err := s.postgres.Pool.QueryRow(ctx, query, req.GetKey()).Scan(&userId, &email)
	if err != nil {
		keyPrefix := req.GetKey()
		if len(keyPrefix) > 8 {
			keyPrefix = keyPrefix[:8]
		}
		slog.LogAttrs(ctx, slog.LevelError, "error validating key",
			slog.String("key_prefix", keyPrefix),
			slog.String("error", err.Error()))
		return nil, ErrInternal
	}

	if userId == nil || email == nil {
		return nil, ErrKeyNotFound
	}

	return &auth.LoginResponse{UserId: int32(*userId), UserEmail: *email}, nil
}
