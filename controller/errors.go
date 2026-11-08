package controller

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal          = status.Error(codes.Internal, "Internal server error")
	ErrUserNotFound      = status.Error(codes.NotFound, "User not found")
	ErrProjectNotFound   = status.Error(codes.NotFound, "Project not found")
	ErrUnauthenticated   = status.Error(codes.Unauthenticated, "Invalid credentials")
	ErrAccountDisabled   = status.Error(codes.Unauthenticated, "Account disabled")
	ErrTempExpired       = status.Error(codes.Unauthenticated, "Temporary password expired. Ask the administrator for a new one.")
	ErrInvalidKey        = status.Error(codes.NotFound, "Invalid API key")
	ErrKeyAccess         = status.Error(codes.PermissionDenied, "API keys require access to Manager")
	ErrAdminRequired     = status.Error(codes.PermissionDenied, "Administrator access required")
	ErrWrongPassword     = status.Error(codes.InvalidArgument, "Current password is incorrect")
	ErrInvalidRecovery   = status.Error(codes.InvalidArgument, "Invalid email or recovery code")
	ErrSamePassword      = status.Error(codes.InvalidArgument, "New password must differ from the current one")
	ErrAdminTarget       = status.Error(codes.InvalidArgument, "This action is not allowed on the administrator account")
	ErrUnknownProject    = status.Error(codes.InvalidArgument, "One or more projects do not exist")
	ErrEmailTaken        = status.Error(codes.AlreadyExists, "A user with this email already exists")
	ErrSlugTaken         = status.Error(codes.AlreadyExists, "A project with this slug already exists")
	ErrNoPendingPassword = status.Error(codes.FailedPrecondition, "Password is already set")
)
