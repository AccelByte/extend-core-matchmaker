// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package envelope

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/AccelByte/extend-core-matchmaker/pkg/common"
)

const (
	traceIdLogField = "traceID"
	tracerName      = "mm-server"

	ServerNameTag  = "ags.matchmakingv2.server_name"
	TeamMembersTag = "ags.matchmakingv2.team_members"

	abTraceIdLogField = "abTraceID"
)

func ChildScopeFromRemoteScope(ctx context.Context, name string) *Scope {
	tracer := otel.Tracer(tracerName)
	tracerCtx, span := tracer.Start(ctx, name)
	traceID := span.SpanContext().TraceID().String()
	if traceID == "" || len(traceID) != 32 {
		traceID = common.GenerateUUID()
	}

	return &Scope{
		Ctx:     tracerCtx,
		TraceID: traceID,
		span:    span,
		Log:     logrus.WithField(traceIdLogField, traceID),
	}
}

func NewRootScope(rootCtx context.Context, name string, abTraceID string) *Scope {
	tracer := otel.Tracer(name)
	ctx, span := tracer.Start(rootCtx, name)

	if abTraceID == "" || len(abTraceID) != 32 {
		abTraceID = common.GenerateUUID()
	}

	scope := &Scope{
		Ctx:     ctx,
		TraceID: abTraceID,
		span:    span,
		Log:     logrus.WithField(traceIdLogField, abTraceID),
	}

	return scope
}

// Scope used as the envelope to combine and transport request-related information by the chain of function calls
type Scope struct {
	Ctx     context.Context
	TraceID string
	span    oteltrace.Span
	Log     *logrus.Entry
}

// SetLogger allows for setting a different logger than the default std logger. This is mostly useful for testing.
func (s *Scope) SetLogger(logger *logrus.Logger) {
	s.Log = logger.WithField(abTraceIdLogField, s.TraceID)
}

// Finish finishes current scope
func (s *Scope) Finish() {
	s.span.End()
}

// NewChildScope creates new child Scope.
func (s *Scope) NewChildScope(name string) *Scope {
	tracer := s.span.TracerProvider().Tracer(tracerName)
	ctx, span := tracer.Start(s.Ctx, name)

	return &Scope{
		Ctx:     ctx,
		TraceID: s.TraceID,
		span:    span,
		Log:     s.Log,
	}
}

// SetAttributes adds attributes onto a span based on the value object type
func (s *Scope) SetAttributes(key string, value interface{}) {
	switch v := value.(type) {
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case []bool:
		s.span.SetAttributes(attribute.BoolSlice(key, v))
	case []string:
		s.span.SetAttributes(attribute.StringSlice(key, v))
	case []int:
		s.span.SetAttributes(attribute.IntSlice(key, v))
	case []int64:
		s.span.SetAttributes(attribute.Int64Slice(key, v))
	case []float64:
		s.span.SetAttributes(attribute.Float64Slice(key, v))
	case time.Duration:
		s.span.SetAttributes(attribute.Int(key, int(v.Seconds())))
	case time.Time:
		s.span.SetAttributes(attribute.String(key, v.Format(time.RFC1123Z)))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}
