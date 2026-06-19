package controller

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal        = status.Error(codes.Internal, "Internal server error")
	ErrInternalCreate  = status.Error(codes.Internal, "Error creating resource")
	ErrInternalUpdate  = status.Error(codes.Internal, "Error updating resource")
	ErrInternalDelete  = status.Error(codes.Internal, "Error deleting resource")
	ErrNotFound        = status.Error(codes.NotFound, "Not found")
	ErrUserNotFound    = status.Error(codes.NotFound, "User not found")
	ErrUnauthenticated = status.Error(codes.Unauthenticated, "Invalid credentials")
	ErrKeyNotFound     = status.Error(codes.NotFound, "Key not found")
)
