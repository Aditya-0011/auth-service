package controller

import (
	"auth/db"
	"auth/utils"
	"context"
	"log/slog"

	auth "github.com/Aditya-0011/common/contracts/go/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Errorf(codes.Internal, "internal server error")
	}

	if count == 0 {
		hashedPassword, err := utils.GenerateHash(req.GetPassword())
		if err != nil {
			slog.Error("error generating hash", "error", err)
			return nil, status.Errorf(codes.Internal, "error generating hash")
		}

		var (
			newId   int
			outcode int8
		)

		err = s.postgres.Pool.QueryRow(ctx, "select * from account.create($1, $2)", req.GetEmail(), hashedPassword).Scan(&newId, &outcode)
		if err != nil {
			slog.Error("error creating user", "error", err)
			return nil, status.Errorf(codes.Internal, "error creating user")
		}

		switch outcode {
		case -1:
			return &auth.LoginResponse{UserId: int32(newId), UserEmail: req.GetEmail()}, nil
		case 0: //currently only supports single user - Aditya Punmiya
			return nil, status.Errorf(codes.NotFound, "user not found") //return nil, status.Errorf(codes.AlreadyExists, "user already exists")
		case 1:
			return nil, status.Errorf(codes.NotFound, "user not found")
		default:
			return nil, status.Errorf(codes.Internal, "error creating user")
		}

	} else {
		if id == nil || password == nil {
			slog.Error("invalid email provided")
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}

		match, err := utils.ComparePassword(req.GetPassword(), *password)
		if err != nil {
			slog.Error("error verifying password", "error", err)
			return nil, status.Errorf(codes.Internal, "error verifying password")
		}

		if !match {
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}

		return &auth.LoginResponse{UserId: int32(*id), UserEmail: req.GetEmail()}, nil
	}
}
