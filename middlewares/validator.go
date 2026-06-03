package middlewares

import (
	"context"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func ValidationInterceptor(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		if msg, ok := req.(protoreflect.ProtoMessage); ok {
			if err := validator.Validate(msg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
			}
		}

		return handler(ctx, req)
	}
}
