package controller

import (
	"context"
	"errors"
	"time"

	"auth/internal/secret"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) requireKeyAccess(ctx context.Context, userID int32) error {
	var allowed bool
	if err := s.pool.QueryRow(ctx, `select account.can_use_api_key($1)`, userID).Scan(&allowed); err != nil {
		return internal(ctx, "checking api key access", err)
	}
	if !allowed {
		return ErrKeyAccess
	}
	return nil
}

func (s *Server) ValidateKey(ctx context.Context, req *auth.ValidateKeyRequest) (*auth.ApiKeyIdentity, error) {
	var (
		userID  int32
		origins []string
		allowed bool
	)
	err := s.pool.QueryRow(ctx,
		`select c.id, c.key_origins, account.can_use_api_key(c.id) from account.creds c where c.key_hash = $1`,
		secret.Hash(req.GetKey()),
	).Scan(&userID, &origins, &allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidKey
	}
	if err != nil {
		return nil, internal(ctx, "validating api key", err)
	}
	if !allowed {
		return nil, ErrInvalidKey
	}

	id, err := s.loadIdentity(ctx, userID)
	if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrAccountDisabled) {
		return nil, ErrInvalidKey
	}
	if err != nil {
		return nil, err
	}
	return &auth.ApiKeyIdentity{Identity: id, Origins: origins}, nil
}

func (s *Server) GetKey(ctx context.Context, req *auth.UserRequest) (*auth.GetKeyResponse, error) {
	if err := s.requireKeyAccess(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	var (
		hint      *string
		updatedAt time.Time
		origins   []string
	)
	err := s.pool.QueryRow(ctx, `select key_hint, updatedAt, key_origins from account.creds where id = $1`, req.GetUserId()).
		Scan(&hint, &updatedAt, &origins)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, internal(ctx, "reading key", err)
	}

	res := &auth.GetKeyResponse{Origins: origins}
	if hint != nil {
		res.KeyHint = *hint
		res.UpdatedAt = timestamppb.New(updatedAt)
	}
	return res, nil
}

func (s *Server) RotateKey(ctx context.Context, req *auth.UserRequest) (*auth.RotateKeyResponse, error) {
	key, err := secret.NewAPIKey()
	if err != nil {
		return nil, internal(ctx, "generating api key", err)
	}

	var (
		previousHash *string
		updatedAt    *time.Time
		outcode      int16
	)
	err = s.pool.QueryRow(ctx,
		`select previous_key_hash, updated_at, outcode from account.rotate_key($1, $2, $3)`,
		req.GetUserId(), key.Hash, key.Hint,
	).Scan(&previousHash, &updatedAt, &outcode)
	if err != nil {
		return nil, internal(ctx, "rotating key", err)
	}

	switch outcode {
	case outSuccess:
		res := &auth.RotateKeyResponse{Key: key.Plaintext}
		if previousHash != nil {
			res.PreviousKeyHash = *previousHash
		}
		if updatedAt != nil {
			res.UpdatedAt = timestamppb.New(*updatedAt)
		}
		return res, nil
	case outNotAllowed:
		return nil, ErrKeyAccess
	default:
		return nil, ErrInternal
	}
}

func (s *Server) RevokeKey(ctx context.Context, req *auth.UserRequest) (*auth.RevokeKeyResponse, error) {
	var (
		previousHash *string
		outcode      int16
	)
	err := s.pool.QueryRow(ctx, `select previous_key_hash, outcode from account.revoke_key($1)`, req.GetUserId()).
		Scan(&previousHash, &outcode)
	if err != nil {
		return nil, internal(ctx, "revoking key", err)
	}

	switch outcode {
	case outSuccess:
		res := &auth.RevokeKeyResponse{}
		if previousHash != nil {
			res.PreviousKeyHash = *previousHash
		}
		return res, nil
	case outNotFound:
		return nil, ErrUserNotFound
	default:
		return nil, ErrInternal
	}
}

func (s *Server) SetKeyOrigins(ctx context.Context, req *auth.SetKeyOriginsRequest) (*auth.KeyOriginsResponse, error) {
	origins := req.GetOrigins()
	if origins == nil {
		origins = []string{}
	}

	var outcode int16
	if err := s.pool.QueryRow(ctx, `select outcode from account.set_key_origins($1, $2)`, req.GetUserId(), origins).
		Scan(&outcode); err != nil {
		return nil, internal(ctx, "saving key origins", err)
	}

	switch outcode {
	case outSuccess:
		return &auth.KeyOriginsResponse{Origins: origins}, nil
	case outNotAllowed:
		return nil, ErrKeyAccess
	default:
		return nil, ErrInternal
	}
}

func (s *Server) ListKeyOrigins(ctx context.Context, _ *auth.ListKeyOriginsRequest) (*auth.KeyOriginsResponse, error) {
	rows, err := s.pool.Query(ctx, `
		select distinct o.origin
		from account.creds c, unnest(c.key_origins) as o(origin)
		where c.key_hash is not null and account.can_use_api_key(c.id)
		order by o.origin`)
	if err != nil {
		return nil, internal(ctx, "listing key origins", err)
	}
	origins, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, internal(ctx, "listing key origins", err)
	}
	return &auth.KeyOriginsResponse{Origins: origins}, nil
}
