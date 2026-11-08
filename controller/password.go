package controller

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"auth/internal/password"
	"auth/internal/secret"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) CompleteFirstLogin(ctx context.Context, req *auth.CompleteFirstLoginRequest) (*auth.RecoveryCodesResponse, error) {
	hashed, err := password.Hash(ctx, req.GetNewPassword())
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}
	codes, hashes, err := secret.NewRecoveryCodes()
	if err != nil {
		return nil, internal(ctx, "generating recovery codes", err)
	}

	outcode, err := s.callOutcode(ctx, "completing first login",
		`select outcode from account.complete_first_login($1, $2, $3)`, req.GetUserId(), hashed, hashes)
	if err != nil {
		return nil, err
	}
	switch outcode {
	case outSuccess:
		return &auth.RecoveryCodesResponse{Codes: codes}, nil
	case outNotFound:
		return nil, ErrNoPendingPassword
	default:
		return nil, ErrInternal
	}
}

func (s *Server) ChangePassword(ctx context.Context, req *auth.ChangePasswordRequest) (*auth.SimpleResponse, error) {
	if req.GetCurrentPassword() == req.GetNewPassword() {
		return nil, ErrSamePassword
	}
	if err := s.checkPassword(ctx, req.GetUserId(), req.GetCurrentPassword()); err != nil {
		return nil, err
	}

	hashed, err := password.Hash(ctx, req.GetNewPassword())
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}
	outcode, err := s.callOutcode(ctx, "changing password", `select outcode from account.set_password($1, $2)`, req.GetUserId(), hashed)
	if err != nil {
		return nil, err
	}
	if outcode != outSuccess {
		return nil, ErrUserNotFound
	}
	return &auth.SimpleResponse{Message: "Password changed"}, nil
}

func (s *Server) ResetPassword(ctx context.Context, req *auth.ResetPasswordRequest) (*auth.Identity, error) {
	hashed, err := password.Hash(ctx, req.GetNewPassword())
	if err != nil {
		return nil, internal(ctx, "hashing password", err)
	}

	var (
		id      int32
		outcode int16
	)
	err = s.pool.QueryRow(ctx, `select id, outcode from account.reset_password($1, $2, $3)`,
		normalizeEmail(req.GetEmail()), secret.HashRecoveryCode(req.GetRecoveryCode()), hashed,
	).Scan(&id, &outcode)
	if err != nil {
		return nil, internal(ctx, "resetting password", err)
	}

	switch outcode {
	case outSuccess:
		slog.LogAttrs(ctx, slog.LevelWarn, "password reset with a recovery code", slog.Int("userId", int(id)))
		return s.loadIdentity(ctx, id)
	case outNotFound:
		return nil, ErrInvalidRecovery
	default:
		return nil, ErrInternal
	}
}

func (s *Server) GetRecoveryCodeStatus(ctx context.Context, req *auth.UserRequest) (*auth.RecoveryCodeStatusResponse, error) {
	var (
		remaining   int32
		generatedAt *time.Time
	)
	err := s.pool.QueryRow(ctx,
		`select count(*), max(createdAt) from account.recovery_code where userId = $1`, req.GetUserId(),
	).Scan(&remaining, &generatedAt)
	if err != nil {
		return nil, internal(ctx, "reading recovery code status", err)
	}

	res := &auth.RecoveryCodeStatusResponse{Remaining: remaining}
	if generatedAt != nil {
		res.GeneratedAt = timestamppb.New(*generatedAt)
	}
	return res, nil
}

func (s *Server) GenerateRecoveryCodes(ctx context.Context, req *auth.PasswordConfirmRequest) (*auth.RecoveryCodesResponse, error) {
	if err := s.checkPassword(ctx, req.GetUserId(), req.GetPassword()); err != nil {
		return nil, err
	}

	codes, hashes, err := secret.NewRecoveryCodes()
	if err != nil {
		return nil, internal(ctx, "generating recovery codes", err)
	}
	outcode, err := s.callOutcode(ctx, "storing recovery codes",
		`select outcode from account.replace_recovery_codes($1, $2)`, req.GetUserId(), hashes)
	if err != nil {
		return nil, err
	}
	if outcode != outSuccess {
		return nil, ErrUserNotFound
	}
	return &auth.RecoveryCodesResponse{Codes: codes}, nil
}

func (s *Server) DeleteRecoveryCodes(ctx context.Context, req *auth.PasswordConfirmRequest) (*auth.SimpleResponse, error) {
	if err := s.checkPassword(ctx, req.GetUserId(), req.GetPassword()); err != nil {
		return nil, err
	}

	outcode, err := s.callOutcode(ctx, "deleting recovery codes",
		`select outcode from account.delete_recovery_codes($1)`, req.GetUserId())
	if err != nil {
		return nil, err
	}
	if outcode != outSuccess {
		return nil, ErrUserNotFound
	}
	return &auth.SimpleResponse{Message: "Backup codes deleted"}, nil
}

func (s *Server) checkPassword(ctx context.Context, userID int32, pass string) error {
	var stored []byte
	err := s.pool.QueryRow(ctx, `select password from account.creds where id = $1 and status = 1`, userID).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return internal(ctx, "reading password", err)
	}

	match, _, err := password.Verify(ctx, pass, stored)
	if err != nil {
		return internal(ctx, "verifying password", err)
	}
	if !match {
		return ErrWrongPassword
	}
	return nil
}
