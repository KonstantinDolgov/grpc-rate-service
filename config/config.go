package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	GRPCPort      string `env:"GRPC_PORT" envDefault:"50051"`
	KuCoinBaseURL string `env:"KUCOIN_BASE_URL" envDefault:"https://api.kucoin.com"`

	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `env:"DB_PORT"`
	DBUser     string `env:"DB_USER"`
	DBPassword string `env:"DB_PASSWORD"`
	DBName     string `env:"DB_NAME"`
	DBSSLMode  string `env:"DB_SSL_MODE" envDefault:"disable"`

	LogLevel          string `env:"LOG_LEVEL" envDefault:"DEBUG"`
	EnableDebugServer bool   `env:"ENABLE_DEBUG_SERVER" envDefault:"true"`
	DebugServerAddr   string `env:"DEBUG_SERVER_ADDR" envDefault:"0.0.0.0:8182"`

	ServiceName    string `env:"SERVICE_NAME" envDefault:"rate-service"`
	ServiceVersion string `env:"SERVICE_VERSION" envDefault:"1.0.0"`
	Environment    string `env:"ENVIRONMENT" envDefault:"development"`

	EnableTracing bool   `env:"ENABLE_TRACING" envDefault:"true"`
	OTLPEndpoint  string `env:"OTLP_ENDPOINT" envDefault:"localhost:4317"`

	EnableMetrics   bool   `env:"ENABLE_METRICS" envDefault:"true"`
	MetricsHTTPAddr string `env:"METRICS_HTTP_ADDR" envDefault:"0.0.0.0:9090"`
}

func ReadConfig() (*Config, error) {
	config := Config{}

	err := env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("read config error: %w", err)
	}

	flag.StringVar(&config.GRPCPort, "grpc-port", config.GRPCPort, "GRPC server port")
	flag.StringVar(&config.DBHost, "db-host", config.DBHost, "Database host")
	flag.StringVar(&config.DBPort, "db-port", config.DBPort, "Database port")
	flag.StringVar(&config.DBUser, "db-user", config.DBUser, "Database user")
	flag.StringVar(&config.DBPassword, "db-password", config.DBPassword, "Database password")
	flag.StringVar(&config.DBName, "db-name", config.DBName, "Database name")
	flag.StringVar(&config.DBSSLMode, "db-sslmode", config.DBSSLMode, "Database SSL mode")
	flag.StringVar(&config.KuCoinBaseURL, "kucoin-base-url", config.KuCoinBaseURL, "KuCoin API base URL")

	flag.BoolVar(&config.EnableTracing, "enable-tracing",
		config.EnableTracing, "Enable OpenTelemetry tracing")
	flag.StringVar(&config.OTLPEndpoint, "otlp-endpoint",
		config.OTLPEndpoint, "OpenTelemetry collector endpoint")
	flag.BoolVar(&config.EnableMetrics, "enable-metrics",
		config.EnableMetrics, "Enable Prometheus metrics")
	flag.StringVar(&config.MetricsHTTPAddr, "metrics-http-addr",
		config.MetricsHTTPAddr, "Prometheus metrics HTTP server address")

	flag.Parse()

	return &config, err
}

func (c *Config) GetDBConnString() string {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)

	return connStr
}
