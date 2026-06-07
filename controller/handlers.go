package controller

import (
	"auth/db"
	"auth/utils"
	"context"
	"log/slog"
	"time"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	ctx, cancel := context.WithTimeout(c, utils.TimeoutDuration)
	defer cancel()

	var (
		count    int
		id       *int
		password *string
	)

	query := `
		with total as (select count(*) as cnt from account.creds)
		select total.cnt, c.id, c.password 
		from total 
		left join account.creds c on c.email = $1
	`
	err := s.postgres.Pool.QueryRow(ctx, query, req.GetEmail()).Scan(&count, &id, &password)
	if err != nil {
		slog.Error("error querying database", "error", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if count == 0 {
		hashedPassword, err := utils.GenerateHash(req.GetPassword())
		if err != nil {
			slog.Error("error generating hash", "error", err)
			return nil, status.Errorf(codes.Internal, "Error generating hash")
		}

		var (
			newId   int
			outcode int8
		)

		err = s.postgres.Pool.QueryRow(ctx, "select * from account.create($1, $2)", req.GetEmail(), hashedPassword).Scan(&newId, &outcode)
		if err != nil {
			slog.Error("error creating user", "error", err)
			return nil, status.Errorf(codes.Internal, "Error creating user")
		}

		switch outcode {
		case -1:
			return &auth.LoginResponse{UserId: int32(newId), UserEmail: req.GetEmail()}, nil
		case 0: //currently only supports single user - Aditya Punmiya
			return nil, status.Errorf(codes.NotFound, "User not found") //return nil, status.Errorf(codes.AlreadyExists, "user already exists")
		case 1:
			return nil, status.Errorf(codes.NotFound, "User not found")
		default:
			return nil, status.Errorf(codes.Internal, "Error creating user")
		}

	} else {
		if id == nil || password == nil {
			slog.Error("invalid email provided")
			return nil, status.Errorf(codes.Unauthenticated, "Invalid credentials")
		}

		match, err := utils.ComparePassword(req.GetPassword(), *password)
		if err != nil {
			slog.Error("error verifying password", "error", err)
			return nil, status.Errorf(codes.Internal, "Error verifying password")
		}

		if !match {
			return nil, status.Errorf(codes.Unauthenticated, "Invalid credentials")
		}

		return &auth.LoginResponse{UserId: int32(*id), UserEmail: req.GetEmail()}, nil
	}
}

func (s *server) GetKey(c context.Context, req *auth.KeyRequest) (*auth.GetKeyResponse, error) {
	ctx, cancel := context.WithTimeout(c, utils.TimeoutDuration)
	defer cancel()

	var (
		key       string
		updatedAt time.Time
	)

	query := `
		select key, updatedAt from account.creds where id = $1
	`

	err := s.postgres.Pool.QueryRow(ctx, query, req.GetUserId()).Scan(&key, &updatedAt)
	if err != nil {
		slog.Error("error querying database", "error", err)
		return nil, status.Errorf(codes.Internal, "Internal server error")
	}

	if key == "" {
		return nil, status.Errorf(codes.NotFound, "Key not found")
	}

	return &auth.GetKeyResponse{
		Key:       key,
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

func (s *server) RotateKey(c context.Context, req *auth.KeyRequest) (*auth.SimpleResponse, error) {
	ctx, cancel := context.WithTimeout(c, utils.TimeoutDuration)
	defer cancel()

	var outcode int8

	query := `
		select outcode from account.rotate_key(@id)
	`

	queryParams := pgx.NamedArgs{
		"id": req.GetUserId(),
	}

	err := s.postgres.Pool.QueryRow(ctx, query, queryParams).Scan(&outcode)
	if err != nil {
		slog.Error("error rotating key", "userId", req.GetUserId(), "error", err)
		return nil, status.Errorf(codes.Internal, "Error rotating key")
	}

	switch outcode {
	case -1:
		return &auth.SimpleResponse{Message: "Key rotated successfully"}, nil
	case 1:
		return nil, status.Errorf(codes.NotFound, "User not found")
	default:
		return nil, status.Errorf(codes.Internal, "Error rotating user")
	}
}

func (s *server) ValidateKey(c context.Context, req *auth.ValidateKeyRequest) (*auth.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(c, utils.TimeoutDuration)
	defer cancel()

	var (
		userId *int
		email  *string
	)

	query := `
		select id, email from account.creds where key = $1
	`
	err := s.postgres.Pool.QueryRow(ctx, query, req.GetKey()).Scan(&userId, &email)
	if err != nil {
		slog.Error("error validating key", "key", req.GetKey(), "error", err)
		return nil, status.Errorf(codes.Internal, "Error validating key")
	}

	if userId == nil || email == nil {
		return nil, status.Errorf(codes.NotFound, "Key not found")
	}

	return &auth.LoginResponse{UserId: int32(*userId), UserEmail: *email}, nil

}
