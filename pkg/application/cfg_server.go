package application

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/encoding/yaml"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"net"
	"net/http"
)

type ConfigServerSettings struct {
	Port int `cfg:"port" default:"8070"`
}

type ConfigServer struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	config   cfg.Config
	logger   log.Logger
	server   *http.Server
	settings *ConfigServerSettings
}

func NewConfigServer() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		settings := &ConfigServerSettings{}
		config.UnmarshalKey("cfg.server", settings)

		server := &ConfigServer{
			config:   config,
			logger:   logger.WithChannel("config-server"),
			server:   &http.Server{},
			settings: settings,
		}

		return server, nil
	}
}

func (s *ConfigServer) Run(ctx context.Context) error {
	var err error
	var listener net.Listener
	var addr = fmt.Sprintf(":%d", s.settings.Port)

	if listener, err = net.Listen("tcp", addr); err != nil {
		return fmt.Errorf("can not listen on address %s: %w", addr, err)
	}

	s.logger.Info("serving config on address %s", listener.Addr())

	handler := http.NewServeMux()
	handler.HandleFunc("/", s.handleRead)

	s.server.Handler = handler
	go s.waitForStop(ctx)

	if err = s.server.Serve(listener); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *ConfigServer) handleRead(writer http.ResponseWriter, request *http.Request) {
	var err error
	var bytes []byte
	var settings = s.config.AllSettings()
	var marshaller = yaml.Marshal

	format := request.URL.Query().Get("format")

	switch format {
	case "json":
		marshaller = json.Marshal
	}

	if bytes, err = marshaller(settings); err != nil {
		s.logger.Warn("can not marshal config %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = writer.Write(bytes); err != nil {
		s.logger.Warn("can not write config %s", err.Error())
	}
}

func (s *ConfigServer) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := s.server.Close()

	if err != nil {
		s.logger.Error("could not close config server: %w", err)
	}
}
