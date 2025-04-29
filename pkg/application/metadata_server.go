package application

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	dx.RegisterRandomizablePortSetting("appctx.metadata.server.port")
}

type MetadataServerSettings struct {
	Port int `cfg:"port" default:"8070"`
}

type MetadataServer struct {
	kernel.EssentialBackgroundModule
	kernel.EssentialStage

	config   cfg.Config
	logger   log.Logger
	server   *http.Server
	settings *MetadataServerSettings
}

func NewMetadataServer(_ context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	settings := &MetadataServerSettings{}
	config.UnmarshalKey("appctx.metadata.server", settings)

	server := &MetadataServer{
		config:   config,
		logger:   logger.WithChannel("metadata-server"),
		server:   &http.Server{},
		settings: settings,
	}

	return server, nil
}

func (s *MetadataServer) Run(ctx context.Context) error {
	var err error
	var metadata *appctx.Metadata
	var listener net.Listener

	if metadata, err = appctx.ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access metadata: %w", err)
	}

	addr := fmt.Sprintf(":%d", s.settings.Port)

	if listener, err = net.Listen("tcp", addr); err != nil {
		return fmt.Errorf("can not listen on address %s: %w", addr, err)
	}

	s.logger.Info("serving metadata on address %s", listener.Addr())

	handler := http.NewServeMux()
	handler.HandleFunc("/", s.handleMetadata(metadata))
	handler.HandleFunc("/config", s.handleConfig)
	handler.HandleFunc("/memory", s.handleMemory)

	s.server.Handler = handler
	go s.waitForStop(ctx)

	if err = s.server.Serve(listener); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *MetadataServer) handleMetadata(metadata *appctx.Metadata) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var err error
		var bytes []byte

		data := metadata.Msi()

		if bytes, err = json.Marshal(data); err != nil {
			s.logger.Warn("can not marshal metadata %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err = writer.Write(bytes); err != nil {
			s.logger.Warn("can not write config %s", err.Error())
		}
	}
}

func (s *MetadataServer) handleConfig(writer http.ResponseWriter, request *http.Request) {
	settings := s.config.AllSettings()
	s.formattedResponse(writer, request, settings)
}

func (s *MetadataServer) handleMemory(writer http.ResponseWriter, request *http.Request) {
	var memMstats runtime.MemStats

	gc := request.URL.Query().Get("gc")
	switch gc {
	case "true":
		runtime.GC()
	}

	runtime.ReadMemStats(&memMstats)
	s.formattedResponse(writer, request, memMstats)
}

func (s *MetadataServer) formattedResponse(writer http.ResponseWriter, request *http.Request, response interface{}) {
	var err error
	var bytes []byte
	marshaller := yaml.Marshal

	format := request.URL.Query().Get("format")

	switch format {
	case "json":
		marshaller = json.Marshal
	}

	if bytes, err = marshaller(response); err != nil {
		s.logger.Warn("can not marshal response %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = writer.Write(bytes); err != nil {
		s.logger.Warn("can not write response %s", err.Error())
	}
}

func (s *MetadataServer) waitForStop(ctx context.Context) {
	<-ctx.Done()
	err := s.server.Close()
	if err != nil {
		s.logger.Error("could not close config server: %w", err)
	}
}
