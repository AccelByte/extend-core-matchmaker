// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package common

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"github.com/AccelByte/accelbyte-go-sdk/iam-sdk/pkg/iamclientmodels"
	"github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/service/iam"
	"github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/utils/auth/validator"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var Validator validator.AuthTokenValidator

func UnaryAuthServerIntercept(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if !skipCheckAuthorizationMetadata(info.FullMethod) {
		err := checkAuthorizationMetadata(ctx)

		if err != nil {
			return nil, err
		}
	}

	return handler(ctx, req)
}

func StreamAuthServerIntercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if !skipCheckAuthorizationMetadata(info.FullMethod) {
		err := checkAuthorizationMetadata(ss.Context())

		if err != nil {
			return err
		}
	}

	return handler(srv, ss)
}

func skipCheckAuthorizationMetadata(fullMethod string) bool {
	if strings.HasPrefix(fullMethod, "/grpc.reflection.v1alpha.ServerReflection/") {
		return true
	}

	if strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/") {
		return true
	}

	return false
}

func checkAuthorizationMetadata(ctx context.Context) error {
	if Validator == nil {
		return status.Error(codes.Internal, "authorization token validator is not set")
	}

	meta, found := metadata.FromIncomingContext(ctx)

	if !found {
		return status.Error(codes.Unauthenticated, "metadata is missing")
	}

	if _, ok := meta["authorization"]; !ok {
		return status.Error(codes.Unauthenticated, "authorization metadata is missing")
	}

	if len(meta["authorization"]) == 0 {
		return status.Error(codes.Unauthenticated, "authorization metadata length is 0")
	}

	authorization := meta["authorization"][0]
	token := strings.TrimPrefix(authorization, "Bearer ")
	namespace := os.Getenv("AB_NAMESPACE")

	err := Validator.Validate(token, nil, &namespace, nil)

	if err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	return nil
}

func NewTokenValidator(authService iam.OAuth20Service, refreshInterval time.Duration, validateLocally bool) validator.AuthTokenValidator {
	return &iam.TokenValidator{
		AuthService:     authService,
		RefreshInterval: refreshInterval,

		Filter:                nil,
		JwkSet:                nil,
		JwtClaims:             iam.JWTClaims{},
		JwtEncoding:           *base64.URLEncoding.WithPadding(base64.NoPadding),
		PublicKeys:            make(map[string]*rsa.PublicKey),
		LocalValidationActive: validateLocally,
		RevokedUsers:          make(map[string]time.Time),
		Roles:                 make(map[string]*iamclientmodels.ModelRolePermissionResponseV3),
	}
}
