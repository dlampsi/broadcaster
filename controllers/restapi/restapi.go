package restapi

import (
	"a0feed/utils/info"
	"context"
	"fmt"

	"github.com/dlampsi/gsrv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

type Config struct {
	Address string `envconfig:"ADDRESS" default:"0.0.0.0:8080"`
}

type Service struct {
	cfg    *Config
	logger *zap.SugaredLogger
}

type Option func(*Service)

func WithLogger(l *zap.SugaredLogger) Option {
	return func(s *Service) { s.logger = l }
}

func New(opts ...Option) (*Service, error) {
	var cfg Config
	if err := envconfig.Process(info.EnvPrefix, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load configuration from env: %w", err)
	}
	s := &Service{
		cfg:    &cfg,
		logger: zap.NewNop().Sugar(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.logger = s.logger.Named("restapi")

	return s, nil
}

func (s *Service) Serve(ctx context.Context) error {
	srv, err := gsrv.New(s.cfg.Address, gsrv.WithLogger(s.logger))
	if err != nil {
		return fmt.Errorf("Can't init new server: %w", err)
	}
	return srv.ServeHTTP(ctx, s.routes())
}
