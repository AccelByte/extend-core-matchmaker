// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/factory"
	"github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/repository"
	"github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/service/iam"
	sdkAuth "github.com/AccelByte/accelbyte-go-sdk/services-api/pkg/utils/auth"
	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker/defaultmatchmaker"
	matchfunctiongrpc "github.com/AccelByte/extend-core-matchmaker/pkg/pb"
	"github.com/AccelByte/extend-core-matchmaker/pkg/server"
	"github.com/caarlos0/env"
	promgrpc "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/AccelByte/extend-core-matchmaker/pkg/common"
)

const (
	environment     = "production"
	id              = int64(1)
	metricsEndpoint = "/metrics"
	metricsPort     = 8080
	grpcPort        = 6565
)

var (
	serviceName = common.GetEnv("OTEL_SERVICE_NAME", "MatchmakingFunctionGrpcPluginServerGoDocker")
	logLevelStr = common.GetEnv("LOG_LEVEL", logrus.InfoLevel.String())
)

func main() {
	go func() {
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(10)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.Infof("starting app server.")

	logrusLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		logrusLevel = logrus.InfoLevel
	}
	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logrusLevel)

	loggingOptions := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall, logging.PayloadReceived, logging.PayloadSent),
		logging.WithFieldsFromContext(func(ctx context.Context) logging.Fields {
			if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
				return logging.Fields{"traceID", span.TraceID().String()}
			}

			return nil
		}),
		logging.WithLevels(logging.DefaultClientCodeToLevel),
		logging.WithDurationField(logging.DurationToDurationField),
	}

	srvMetrics := promgrpc.NewServerMetrics()
	unaryServerInterceptors := []grpc.UnaryServerInterceptor{
		srvMetrics.UnaryServerInterceptor(),
		logging.UnaryServerInterceptor(common.InterceptorLogger(logrusLogger), loggingOptions...),
	}
	streamServerInterceptors := []grpc.StreamServerInterceptor{
		srvMetrics.StreamServerInterceptor(),
		logging.StreamServerInterceptor(common.InterceptorLogger(logrusLogger), loggingOptions...),
	}

	// Preparing the IAM authorization
	var tokenRepo repository.TokenRepository = sdkAuth.DefaultTokenRepositoryImpl()
	var configRepo repository.ConfigRepository = sdkAuth.DefaultConfigRepositoryImpl()
	var refreshRepo repository.RefreshTokenRepository = &sdkAuth.RefreshTokenImpl{RefreshRate: 1.0, AutoRefresh: true}

	oauthService := iam.OAuth20Service{
		Client:                 factory.NewIamClient(configRepo),
		TokenRepository:        tokenRepo,
		RefreshTokenRepository: refreshRepo,
		ConfigRepository:       configRepo,
	}

	if strings.ToLower(common.GetEnv("PLUGIN_GRPC_SERVER_AUTH_ENABLED", "true")) == "true" {
		refreshInterval := common.GetEnvInt("REFRESH_INTERVAL", 600)
		common.Validator = common.NewTokenValidator(oauthService, time.Duration(refreshInterval)*time.Second, true)
		common.Validator.Initialize(ctx)

		unaryServerInterceptors = append(unaryServerInterceptors, common.UnaryAuthServerIntercept)
		streamServerInterceptors = append(streamServerInterceptors, common.StreamAuthServerIntercept)
		logrus.Infof("added auth interceptors")
	}

	// Create gRPC Server
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(unaryServerInterceptors...),
		grpc.ChainStreamInterceptor(streamServerInterceptors...),
	)

	cfg := &config.Config{}
	err = env.Parse(cfg)
	if err != nil {
		logrus.Fatal("unable to parse environment variables: ", err)
	}

	// Enable gRPC Reflection
	reflection.Register(grpcServer)
	logrus.Infof("gRPC reflection enabled")

	// Enable gRPC Health Check
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())

	// Add go runtime metrics and process collectors.
	srvMetrics.InitializeMetrics(grpcServer)
	promRegistry := prometheus.NewRegistry()
	promRegistry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		srvMetrics,
	)

	matchMaker := defaultmatchmaker.New(cfg)
	matchfunctiongrpc.RegisterMatchFunctionServer(grpcServer, &server.MatchFunctionServer{
		UnimplementedMatchFunctionServer: matchfunctiongrpc.UnimplementedMatchFunctionServer{},
		MM:                               matchMaker,
	})

	go func() {
		http.Handle(metricsEndpoint, promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
		logrus.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil))
	}()
	logrus.Printf("prometheus metrics served at :8080/metrics")

	logrus.Infof("listening to grpc port.")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)

		return
	}

	logrus.Printf("gRPC server listening at %v", lis.Addr())

	logrus.Infof("listening...")
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("failed to serve: %v", err)

			return
		}
	}()

	logrus.Infof("starting init provider.")

	// Save Tracer Provider
	tracerProvider, err := common.NewTracerProvider(serviceName, environment, id)
	if err != nil {
		logrus.Fatalf("failed to create tracer provider: %v", err)

		return
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tracerProvider)
	// Register the B3 propagator globally.
	p := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		p,
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			logrus.Fatal(err)
		}
	}(ctx)

	flag.Parse()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logrus.Infof("signal received")
}
