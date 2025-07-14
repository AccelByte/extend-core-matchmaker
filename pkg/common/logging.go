// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package common

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/sirupsen/logrus"
)

// InterceptorLogger adapts logrus logger to interceptor logger.
// This code is referenced from https://github.com/grpc-ecosystem/go-grpc-middleware/
func InterceptorLogger(logger logrus.FieldLogger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		logrusFields := make(map[string]any, len(fields))
		iterator := logging.Fields(fields).Iterator()
		for iterator.Next() {
			fieldName, fieldValue := iterator.At()
			logrusFields[fieldName] = fieldValue
		}
		logger = logger.WithFields(logrusFields)

		switch lvl {
		case logging.LevelDebug:
			logger.Debug(msg)
		case logging.LevelInfo:
			logger.Info(msg)
		case logging.LevelWarn:
			logger.Warn(msg)
		case logging.LevelError:
			logger.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
