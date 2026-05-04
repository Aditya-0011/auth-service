package controller

import (
	"auth/db"
	"auth/utils"
	"context"
	"log"

	auth "github.com/Aditya-0011/common/protos/contracts/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	server struct {
		auth.UnimplementedAuthServiceServer
		postgres *db.PostgresParams
	}

	AuthControllerParams struct {
		Postgres *db.PostgresParams
	}
)

func AuthController(params AuthControllerParams) *server {
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
	err := s.postgres.Pool.QueryRow(ctx, query, req.Email).Scan(&count, &id, &password)
	if err != nil {
		log.Printf("error querying database: %v", err)
		return nil, status.Errorf(codes.Internal, "internal server error")
	}

	if count == 0 {
		hashedPassword, err := utils.GenerateHash(req.Password)
		if err != nil {
			log.Printf("error generating hash: %v", err)
			return nil, status.Errorf(codes.Internal, "error generating hash")
		}

		var (
			newId   int
			outcode int8
		)

		err = s.postgres.Pool.QueryRow(ctx, "select * from account.create($1, $2)", req.Email, hashedPassword).Scan(&newId, &outcode)
		if err != nil {
			log.Printf("error creating user: %v", err)
			return nil, status.Errorf(codes.Internal, "error creating user")
		}

		switch outcode {
		case -1:
			return &auth.LoginResponse{UserId: int64(newId), UserEmail: req.Email}, nil
		case 0:
			return nil, status.Errorf(codes.AlreadyExists, "user already exists")
		case 1:
			return nil, status.Errorf(codes.NotFound, "user not found")
		default:
			return nil, status.Errorf(codes.Internal, "error creating user")
		}

	} else {
		if id == nil || password == nil {
			log.Printf("invalid email provided")
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}

		match, err := utils.ComparePassword(req.Password, *password)
		if err != nil {
			log.Printf("error verifying password: %v", err)
			return nil, status.Errorf(codes.Internal, "error verifying password")
		}

		if !match {
			return nil, status.Errorf(codes.Unauthenticated, "invalid credentials")
		}

		return &auth.LoginResponse{UserId: int64(*id), UserEmail: req.Email}, nil
	}
}
