//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"grpc_gateway_framework/internal/app"
	"grpc_gateway_framework/internal/conf"
	"grpc_gateway_framework/internal/logger"
)

// service provider
func provideServices() []interface{} {
	return []interface{}{}
}

func initApp(appConfig *conf.AppConfig) (*app.App, func(), error) {
	wire.Build(
		// Extract fields from appConfig
		wire.FieldsOf(new(*conf.AppConfig), "LogConfig", "Port", "Mode"),

		// Component providers
		logger.NewLogger,
		provideServices,
		conf.NewUnaryInterceptors,
		conf.NewAllowedHeaders,
		app.NewApp,
	)
	return nil, nil, nil
}
