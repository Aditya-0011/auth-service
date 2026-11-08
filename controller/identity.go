package controller

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"auth/internal/password"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
)

const identityQuery = `
	select c.id, c.email, c.role, c.status, c.must_change_password,
		array(
			select p.slug from account.project p
			where c.role = 1
			   or (p.status = 1 and exists (
					select 1 from account.project_access a where a.userId = c.id and a.projectId = p.id))
			order by p.slug
		)
	from account.creds c`

func (s *Server) loadIdentity(ctx context.Context, userID int32) (*auth.Identity, error) {
	var (
		id       auth.Identity
		role     int16
		status   int16
		projects []string
	)
	err := s.pool.QueryRow(ctx, identityQuery+" where c.id = $1", userID).
		Scan(&id.UserId, &id.Email, &role, &status, &id.MustChangePassword, &projects)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, internal(ctx, "loading identity", err)
	}
	if status != int16(auth.UserStatus_USER_STATUS_ACTIVE) {
		return nil, ErrAccountDisabled
	}
	id.Role = auth.Role(role)
	id.Projects = projects
	return &id, nil
}

func (s *Server) GetIdentity(ctx context.Context, req *auth.UserRequest) (*auth.Identity, error) {
	return s.loadIdentity(ctx, req.GetUserId())
}

func (s *Server) Login(ctx context.Context, req *auth.LoginRequest) (*auth.Identity, error) {
	email := normalizeEmail(req.GetEmail())

	var (
		admins      int
		id          *int32
		stored      []byte
		status      *int16
		mustChange  *bool
		tempExpires *time.Time
	)
	const query = `
		select (select count(*) from account.creds where role = 1), c.id, c.password, c.status,
			c.must_change_password, c.temp_password_expires_at
		from (select 1) as one
		left join account.creds c on lower(c.email) = $1`
	if err := s.pool.QueryRow(ctx, query, email).Scan(&admins, &id, &stored, &status, &mustChange, &tempExpires); err != nil {
		return nil, internal(ctx, "querying credentials", err)
	}

	if id == nil && admins == 0 {
		return s.bootstrap(ctx, email, req.GetPassword())
	}
	if id == nil {
		password.BurnTime(ctx)
		return nil, ErrUnauthenticated
	}

	match, needsRehash, err := password.Verify(ctx, req.GetPassword(), stored)
	if err != nil {
		return nil, internal(ctx, "verifying password", err)
	}
	if !match {
		return nil, ErrUnauthenticated
	}
	if *status != int16(auth.UserStatus_USER_STATUS_ACTIVE) {
		return nil, ErrAccountDisabled
	}
	if *mustChange && tempExpires != nil && time.Now().After(*tempExpires) {
		return nil, ErrTempExpired
	}

	if needsRehash && !*mustChange {
		if hashed, err := password.Hash(ctx, req.GetPassword()); err == nil {
			if _, err := s.callOutcode(ctx, "upgrading password hash", `select outcode from account.rehash_password($1, $2)`, *id, hashed); err != nil {
				slog.LogAttrs(ctx, slog.LevelWarn, "upgrading password hash failed")
			}
		}
	}
	if _, err := s.callOutcode(ctx, "recording login", `select outcode from account.record_login($1)`, *id); err != nil {
		slog.LogAttrs(ctx, slog.LevelWarn, "recording login failed")
	}

	return s.loadIdentity(ctx, *id)
}

func (s *Server) bootstrap(ctx context.Context, email, pass string) (*auth.Identity, error) {
	if !s.allowBootstrap {
		slog.LogAttrs(ctx, slog.LevelWarn, "login rejected: no administrator exists and bootstrap is disabled")
		password.BurnTime(ctx)
		return nil, ErrUnauthenticated
	}

	hashed, err := password.Hash(ctx, pass)
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}

	var (
		id      int32
		outcode int16
	)
	err = s.pool.QueryRow(ctx, `select id, outcode from account.create_user($1, $2, $3, false, null, '{}')`,
		email, hashed, int16(auth.Role_ROLE_ADMIN)).Scan(&id, &outcode)
	if err != nil {
		return nil, internal(ctx, "creating admin", err)
	}

	switch outcode {
	case outSuccess:
		slog.LogAttrs(ctx, slog.LevelWarn, "administrator bootstrapped; unset AUTH_ALLOW_BOOTSTRAP now")
		if _, err := s.callOutcode(ctx, "recording login", `select outcode from account.record_login($1)`, id); err != nil {
			slog.LogAttrs(ctx, slog.LevelWarn, "recording login failed")
		}
		return s.loadIdentity(ctx, id)
	case outExists:
		return nil, ErrUnauthenticated
	default:
		return nil, ErrInternal
	}
}
