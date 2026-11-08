package controller

import (
	"context"
	"errors"
	"time"

	"auth/internal/password"
	"auth/internal/secret"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const temporaryPasswordTTL = 7 * 24 * time.Hour

const userQuery = `
	select c.id, c.email, c.role, c.status, c.must_change_password,
		array(select a.projectId from account.project_access a where a.userId = c.id order by a.projectId),
		c.createdAt, c.last_login_at
	from account.creds c`

func scanUser(row pgx.Row) (*auth.User, error) {
	var (
		u            auth.User
		role, status int16
		createdAt    time.Time
		lastLogin    *time.Time
	)
	if err := row.Scan(&u.Id, &u.Email, &role, &status, &u.MustChangePassword, &u.ProjectIds, &createdAt, &lastLogin); err != nil {
		return nil, err
	}
	u.Role = auth.Role(role)
	u.Status = auth.UserStatus(status)
	u.CreatedAt = timestamppb.New(createdAt)
	if lastLogin != nil {
		u.LastLoginAt = timestamppb.New(*lastLogin)
	}
	return &u, nil
}

func (s *Server) ListUsers(ctx context.Context, req *auth.UserRequest) (*auth.UsersResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, userQuery+` order by c.role, c.email`)
	if err != nil {
		return nil, internal(ctx, "querying users", err)
	}
	defer rows.Close()

	res := &auth.UsersResponse{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, internal(ctx, "scanning user", err)
		}
		res.Users = append(res.Users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, internal(ctx, "iterating users", err)
	}
	return res, nil
}

func (s *Server) CreateUser(ctx context.Context, req *auth.CreateUserRequest) (*auth.CreateUserResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	temp, err := secret.NewTemporaryPassword()
	if err != nil {
		return nil, internal(ctx, "generating temporary password", err)
	}
	hashed, err := password.Hash(ctx, temp)
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}

	var (
		id      int32
		outcode int16
	)
	err = s.pool.QueryRow(ctx,
		`select id, outcode from account.create_user($1, $2, $3, true, $4, $5)`,
		normalizeEmail(req.GetEmail()), hashed, int16(auth.Role_ROLE_TENANT),
		time.Now().Add(temporaryPasswordTTL), req.GetProjectIds(),
	).Scan(&id, &outcode)
	if err != nil {
		return nil, internal(ctx, "creating user", err)
	}

	switch outcode {
	case outSuccess:
	case outExists:
		return nil, ErrEmailTaken
	case outNotFound:
		return nil, ErrUnknownProject
	default:
		return nil, ErrInternal
	}

	user, err := scanUser(s.pool.QueryRow(ctx, userQuery+` where c.id = $1`, id))
	if err != nil {
		return nil, internal(ctx, "reading created user", err)
	}
	return &auth.CreateUserResponse{User: user, TemporaryPassword: temp}, nil
}

func (s *Server) ResetUserPassword(ctx context.Context, req *auth.TargetUserRequest) (*auth.TemporaryPasswordResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	temp, err := secret.NewTemporaryPassword()
	if err != nil {
		return nil, internal(ctx, "generating temporary password", err)
	}
	hashed, err := password.Hash(ctx, temp)
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}

	outcode, err := s.callOutcode(ctx, "resetting user password",
		`select outcode from account.set_temp_password($1, $2, $3)`,
		req.GetTargetUserId(), hashed, time.Now().Add(temporaryPasswordTTL))
	if err != nil {
		return nil, err
	}
	if outcode != outSuccess {
		return nil, s.targetError(ctx, req.GetTargetUserId())
	}
	return &auth.TemporaryPasswordResponse{TemporaryPassword: temp}, nil
}

func (s *Server) SetUserStatus(ctx context.Context, req *auth.SetUserStatusRequest) (*auth.SimpleResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	outcode, err := s.callOutcode(ctx, "setting user status",
		`select outcode from account.set_user_status($1, $2)`, req.GetTargetUserId(), int16(req.GetStatus()))
	if err != nil {
		return nil, err
	}
	switch outcode {
	case outSuccess:
		if req.GetStatus() == auth.UserStatus_USER_STATUS_DISABLED {
			return &auth.SimpleResponse{Message: "User disabled"}, nil
		}
		return &auth.SimpleResponse{Message: "User enabled"}, nil
	case outNotFound:
		return nil, ErrUserNotFound
	case outNotAllowed:
		return nil, ErrAdminTarget
	default:
		return nil, ErrInternal
	}
}

func (s *Server) SetUserProjects(ctx context.Context, req *auth.SetUserProjectsRequest) (*auth.SimpleResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	outcode, err := s.callOutcode(ctx, "setting user projects",
		`select outcode from account.set_user_projects($1, $2)`, req.GetTargetUserId(), req.GetProjectIds())
	if err != nil {
		return nil, err
	}
	switch outcode {
	case outSuccess:
		return &auth.SimpleResponse{Message: "Project access updated"}, nil
	case outNotFound:
		return nil, s.targetError(ctx, req.GetTargetUserId())
	case outNotAllowed:
		return nil, ErrAdminTarget
	default:
		return nil, ErrInternal
	}
}

func (s *Server) DeleteUser(ctx context.Context, req *auth.TargetUserRequest) (*auth.SimpleResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	outcode, err := s.callOutcode(ctx, "deleting user", `select outcode from account.delete_user($1)`, req.GetTargetUserId())
	if err != nil {
		return nil, err
	}
	switch outcode {
	case outSuccess:
		return &auth.SimpleResponse{Message: "User deleted"}, nil
	case outNotFound:
		return nil, ErrUserNotFound
	case outNotAllowed:
		return nil, ErrAdminTarget
	default:
		return nil, ErrInternal
	}
}

func (s *Server) targetError(ctx context.Context, targetID int32) error {
	var role int16
	err := s.pool.QueryRow(ctx, `select role from account.creds where id = $1`, targetID).Scan(&role)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrUserNotFound
	case err != nil:
		return internal(ctx, "reading target user", err)
	case role == int16(auth.Role_ROLE_ADMIN):
		return ErrAdminTarget
	default:
		return ErrUnknownProject
	}
}
