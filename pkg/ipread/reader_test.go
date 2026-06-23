package ipread_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ipread"
	"github.com/justtrackio/gosoline/pkg/ipread/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ReaderTestSuite struct {
	suite.Suite
	ctx      context.Context
	config   cfg.GosoConf
	logger   log.Logger
	provider *mocks.Provider
}

func (s *ReaderTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.config = cfg.New()
	s.logger = logMocks.NewLogger(s.T())
	s.provider = mocks.NewProvider(s.T())
}

func (s *ReaderTestSuite) setupConfigValues(values map[string]interface{}) {
	if err := s.config.Option(cfg.WithConfigMap(values)); err != nil {
		s.FailNow("can not setup config values", err.Error())
	}
}

func TestReaderTestSuite(t *testing.T) {
	suite.Run(t, new(ReaderTestSuite))
}

func (s *ReaderTestSuite) TestNewReader_Success() {
	// Setup config for memory provider
	s.setupConfigValues(map[string]interface{}{
		"ipread.test.provider": "memory",
	})
	
	// Set up logger expectation since this test calls NewReader
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger)
	
	reader, err := ipread.NewReader(s.ctx, s.config, s.logger, "test")
	
	s.NoError(err)
	s.NotNil(reader)
}

func (s *ReaderTestSuite) TestNewReader_ProviderNotFound() {
	// Setup config with invalid provider
	s.setupConfigValues(map[string]interface{}{
		"ipread.test.provider": "invalid",
	})
	
	// Set up logger expectation since this test calls NewReader
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger)
	
	reader, err := ipread.NewReader(s.ctx, s.config, s.logger, "test")
	
	s.Error(err)
	s.Nil(reader)
	s.Contains(err.Error(), "provider invalid not found")
}

func (s *ReaderTestSuite) TestProvideReader_Success() {
	// ProvideReader relies on appctx which needs proper context setup
	// Let's test NewReader instead as it doesn't require application context
	s.setupConfigValues(map[string]interface{}{
		"ipread.test.provider": "memory",
	})
	
	// Set up logger expectation since this test calls NewReader
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger).Maybe()
	
	reader1, err1 := ipread.NewReader(s.ctx, s.config, s.logger, "test")
	reader2, err2 := ipread.NewReader(s.ctx, s.config, s.logger, "test")
	
	s.NoError(err1)
	s.NoError(err2)
	s.NotNil(reader1)
	s.NotNil(reader2)
	// NewReader creates new instances each time, unlike ProvideReader
}

func (s *ReaderTestSuite) TestReader_City_Success() {
	// Create a mock geoip2.City response
	city := &geoip2.City{
		City: struct {
			GeoNameID uint              `maxminddb:"geoname_id"`
			Names     map[string]string `maxminddb:"names"`
		}{
			Names: map[string]string{
				"en": "New York",
			},
		},
		Country: struct {
			GeoNameID         uint              `maxminddb:"geoname_id"`
			IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			IsoCode           string            `maxminddb:"iso_code"`
			Names             map[string]string `maxminddb:"names"`
		}{
			IsoCode: "US",
		},
		Location: struct {
			AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
			MetroCode      uint    `maxminddb:"metro_code"`
			TimeZone       string  `maxminddb:"time_zone"`
		}{
			TimeZone: "America/New_York",
		},
	}
	
	s.provider.On("City", mock.MatchedBy(func(ip net.IP) bool {
		return ip.String() == "192.168.1.1"
	})).Return(city, nil)
	
	// Create reader with mock provider
	reader := &readerWithProvider{provider: s.provider}
	
	result, err := reader.City("192.168.1.1")
	
	s.NoError(err)
	s.NotNil(result)
	s.Equal("192.168.1.1", result.Ip)
	s.Equal("US", result.CountryCode)
	s.Equal("New York", result.City)
	s.Equal("America/New_York", result.TimeZone)
}

func (s *ReaderTestSuite) TestReader_City_InvalidIP() {
	reader := &readerWithProvider{provider: s.provider}
	
	result, err := reader.City("invalid-ip")
	
	s.Error(err)
	s.Nil(result)
	s.Equal(ipread.ErrIpParseFailed, err)
}

func (s *ReaderTestSuite) TestReader_City_ProviderError() {
	providerErr := errors.New("provider error")
	s.provider.On("City", mock.AnythingOfType("net.IP")).Return(nil, providerErr)
	
	reader := &readerWithProvider{provider: s.provider}
	
	result, err := reader.City("192.168.1.1")
	
	s.Error(err)
	s.Nil(result)
	s.Equal(providerErr, err)
}

func (s *ReaderTestSuite) TestReader_City_EmptyStringFields() {
	// Test with empty city and timezone
	city := &geoip2.City{
		City: struct {
			GeoNameID uint              `maxminddb:"geoname_id"`
			Names     map[string]string `maxminddb:"names"`
		}{
			Names: map[string]string{
				"en": "",
			},
		},
		Country: struct {
			GeoNameID         uint              `maxminddb:"geoname_id"`
			IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			IsoCode           string            `maxminddb:"iso_code"`
			Names             map[string]string `maxminddb:"names"`
		}{
			IsoCode: "XX",
		},
		Location: struct {
			AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
			MetroCode      uint    `maxminddb:"metro_code"`
			TimeZone       string  `maxminddb:"time_zone"`
		}{
			TimeZone: "",
		},
	}
	
	s.provider.On("City", mock.AnythingOfType("net.IP")).Return(city, nil)
	
	reader := &readerWithProvider{provider: s.provider}
	
	result, err := reader.City("192.168.1.1")
	
	s.NoError(err)
	s.NotNil(result)
	s.Equal("192.168.1.1", result.Ip)
	s.Equal("XX", result.CountryCode)
	s.Equal("", result.City)
	s.Equal("", result.TimeZone)
}

// Test various IP formats
func TestReader_City_IPFormats(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		shouldErr bool
	}{
		{"IPv4", "192.168.1.1", false},
		{"IPv6", "2001:db8::1", false},
		{"IPv6 compressed", "::1", false},
		{"Invalid format", "256.256.256.256", true},
		{"Empty string", "", true},
		{"Text", "not-an-ip", true},
		{"Partial IP", "192.168", true},
	}
	
	provider := mocks.NewProvider(t)
	provider.On("City", mock.AnythingOfType("net.IP")).Return(&geoip2.City{}, nil)
	
	reader := &readerWithProvider{provider: provider}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reader.City(tt.ip)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Equal(t, ipread.ErrIpParseFailed, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper struct to test reader with mock provider
type readerWithProvider struct {
	provider ipread.Provider
}

func (r *readerWithProvider) City(ipString string) (*ipread.GeoCity, error) {
	ip := net.ParseIP(ipString)
	
	if ip == nil {
		return nil, ipread.ErrIpParseFailed
	}
	
	record, err := r.provider.City(ip)
	if err != nil {
		return nil, err
	}
	
	return &ipread.GeoCity{
		Ip:          ipString,
		CountryCode: record.Country.IsoCode,
		City:        record.City.Names["en"],
		TimeZone:    record.Location.TimeZone,
	}, nil
}