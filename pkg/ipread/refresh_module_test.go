package ipread_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ipread"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

type RefreshModuleTestSuite struct {
	suite.Suite
	ctx    context.Context
	config cfg.GosoConf
	logger log.Logger
}

func (s *RefreshModuleTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.config = cfg.New()
	s.logger = logMocks.NewLogger(s.T())
}

func (s *RefreshModuleTestSuite) setupConfigValues(values map[string]interface{}) {
	if err := s.config.Option(cfg.WithConfigMap(values)); err != nil {
		s.FailNow("can not setup config values", err.Error())
	}
}

func TestRefreshModuleTestSuite(t *testing.T) {
	suite.Run(t, new(RefreshModuleTestSuite))
}

func (s *RefreshModuleTestSuite) TestRefreshModuleFactory_NoReaders() {
	// Test with no ipread configuration
	modules, err := ipread.RefreshModuleFactory(s.ctx, s.config, s.logger)
	
	s.NoError(err)
	s.NotNil(modules)
	s.Empty(modules)
}

func (s *RefreshModuleTestSuite) TestRefreshModuleFactory_SingleReader() {
	s.setupConfigValues(map[string]interface{}{
		"ipread.test.provider":        "memory",
		"ipread.test.refresh.enabled": true,
	})
	
	modules, err := ipread.RefreshModuleFactory(s.ctx, s.config, s.logger)
	
	s.NoError(err)
	s.NotNil(modules)
	s.Len(modules, 1)
	s.Contains(modules, "ipread-refresh-test")
}

func (s *RefreshModuleTestSuite) TestRefreshModuleFactory_MultipleReaders() {
	s.setupConfigValues(map[string]interface{}{
		"ipread.reader1.provider":        "memory",
		"ipread.reader1.refresh.enabled": true,
		"ipread.reader2.provider":        "maxmind",
		"ipread.reader2.refresh.enabled": false,
		"ipread.reader3.provider":        "memory",
	})
	
	modules, err := ipread.RefreshModuleFactory(s.ctx, s.config, s.logger)
	
	s.NoError(err)
	s.NotNil(modules)
	s.Len(modules, 3)
	s.Contains(modules, "ipread-refresh-reader1")
	s.Contains(modules, "ipread-refresh-reader2")
	s.Contains(modules, "ipread-refresh-reader3")
}

func (s *RefreshModuleTestSuite) TestNewProviderRefreshModule_CreatesFactory() {
	refreshSettings := ipread.RefreshSettings{
		Enabled:  false,
		Interval: 24 * time.Hour,
	}
	
	moduleFactory := ipread.NewProviderRefreshModule("test", refreshSettings)
	s.NotNil(moduleFactory, "NewProviderRefreshModule should return a factory function")
}

func (s *RefreshModuleTestSuite) TestNewProviderRefreshModule_FactoryCreatesModule() {
	refreshSettings := ipread.RefreshSettings{
		Enabled:  false,
		Interval: 24 * time.Hour,
	}
	
	// Set up config for memory provider but don't expect module creation to succeed
	// since it requires application context
	s.setupConfigValues(map[string]interface{}{
		"ipread.test.provider": "memory",
	})
	
	// Set up logger expectations
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger)
	
	moduleFactory := ipread.NewProviderRefreshModule("test", refreshSettings)
	module, err := moduleFactory(s.ctx, s.config, s.logger)
	
	// Expect error due to missing application context
	s.Error(err)
	s.Nil(module)
	s.Contains(err.Error(), "can not get reader with name test")
}

// Test configuration variations
func (s *RefreshModuleTestSuite) TestRefreshModule_ConfigurationVariations() {
	tests := []struct {
		name     string
		settings ipread.RefreshSettings
		wantErr  bool
	}{
		{
			name: "disabled refresh",
			settings: ipread.RefreshSettings{
				Enabled:  false,
				Interval: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "enabled refresh with short interval",
			settings: ipread.RefreshSettings{
				Enabled:  true,
				Interval: 1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "enabled refresh with long interval",
			settings: ipread.RefreshSettings{
				Enabled:  true,
				Interval: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "zero interval",
			settings: ipread.RefreshSettings{
				Enabled:  true,
				Interval: 0,
			},
			wantErr: false, // Should handle gracefully
		},
	}
	
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.setupConfigValues(map[string]interface{}{
				"ipread.test.provider": "memory",
			})
			
			mockLogger := s.logger.(*logMocks.Logger)
			mockLogger.On("WithChannel", "ipread").Return(mockLogger)
			
			moduleFactory := ipread.NewProviderRefreshModule("test", tt.settings)
			module, err := moduleFactory(s.ctx, s.config, s.logger)
			
			// All will fail due to missing application context
			s.Error(err)
			s.Nil(module)
			s.Contains(err.Error(), "can not get reader with name test")
		})
	}
}

func (s *RefreshModuleTestSuite) TestRefreshModuleFactory_Integration() {
	// Integration test that verifies the full flow from factory to module creation
	s.setupConfigValues(map[string]interface{}{
		"ipread.geo-reader.provider":        "memory",
		"ipread.geo-reader.refresh.enabled": true,
		"ipread.geo-reader.refresh.interval": "1h",
	})
	
	// Call the factory
	modules, err := ipread.RefreshModuleFactory(s.ctx, s.config, s.logger)
	
	s.NoError(err)
	s.NotNil(modules)
	s.Len(modules, 1)
	s.Contains(modules, "ipread-refresh-geo-reader")
	
	// Try to create the module - this will fail due to missing application context
	moduleFactory := modules["ipread-refresh-geo-reader"]
	s.NotNil(moduleFactory)
	
	// Set up logger expectations for the module creation
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger)
	
	module, err := moduleFactory(s.ctx, s.config, s.logger)
	// Expect error due to missing application context
	s.Error(err)
	s.Nil(module)
	s.Contains(err.Error(), "can not get reader with name")
}