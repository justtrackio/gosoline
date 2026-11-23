package ipread_test

import (
	"context"
	"net"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ipread"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

type MemoryProviderTestSuite struct {
	suite.Suite
	ctx    context.Context
	config cfg.GosoConf
	logger log.Logger
}

func (s *MemoryProviderTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.config = cfg.New()
	s.logger = logMocks.NewLogger(s.T())
	// Most memory provider tests don't use logger directly, so no need to set expectations
}

func (s *MemoryProviderTestSuite) setupConfigValues(values map[string]interface{}) {
	if err := s.config.Option(cfg.WithConfigMap(values)); err != nil {
		s.FailNow("can not setup config values", err.Error())
	}
}

func TestMemoryProviderTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryProviderTestSuite))
}

func (s *MemoryProviderTestSuite) TestProvideMemoryProvider_Singleton() {
	// Use unique names to avoid conflicts with other tests
	name1 := "test-singleton-unique-1"
	name2 := "test-singleton-unique-1" // Same as name1
	name3 := "test-singleton-unique-2" // Different from name1
	
	provider1 := ipread.ProvideMemoryProvider(name1)
	provider2 := ipread.ProvideMemoryProvider(name2)
	provider3 := ipread.ProvideMemoryProvider(name3)
	
	s.NotNil(provider1)
	s.NotNil(provider2)
	s.NotNil(provider3)
	s.Equal(provider1, provider2, "Same name should return same instance")
	
	// Add different data to verify they are actually different instances
	provider1.AddRecord("192.168.1.1", ipread.MemoryRecord{CountryIso: "US"})
	provider3.AddRecord("192.168.1.1", ipread.MemoryRecord{CountryIso: "GB"})
	
	// Now verify they behave as different instances
	city1, _ := provider1.City(net.ParseIP("192.168.1.1"))
	city3, _ := provider3.City(net.ParseIP("192.168.1.1"))
	
	s.NotEqual(city1.Country.IsoCode, city3.Country.IsoCode, "Different providers should have different data")
}

func (s *MemoryProviderTestSuite) TestNewMemoryProvider() {
	provider, err := ipread.NewMemoryProvider(s.ctx, s.config, s.logger, "test")
	
	s.NoError(err)
	s.NotNil(provider)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_AddRecord_And_City() {
	provider := ipread.ProvideMemoryProvider("test-add-record")
	
	record := ipread.MemoryRecord{
		CountryIso: "US",
		CityName:   "New York",
		TimeZone:   "America/New_York",
	}
	
	provider.AddRecord("192.168.1.1", record)
	
	ip := net.ParseIP("192.168.1.1")
	city, err := provider.City(ip)
	
	s.NoError(err)
	s.NotNil(city)
	s.Equal("US", city.Country.IsoCode)
	s.Equal("New York", city.City.Names["en"])
	s.Equal("America/New_York", city.Location.TimeZone)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_City_NotFound() {
	provider := ipread.ProvideMemoryProvider("test-not-found")
	
	ip := net.ParseIP("192.168.1.1")
	city, err := provider.City(ip)
	
	s.Error(err)
	s.Nil(city)
	s.Equal(ipread.ErrIpNotFound, err)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_City_MultipleRecords() {
	provider := ipread.ProvideMemoryProvider("test-multiple")
	
	// Add multiple records
	provider.AddRecord("192.168.1.1", ipread.MemoryRecord{
		CountryIso: "US",
		CityName:   "New York",
		TimeZone:   "America/New_York",
	})
	
	provider.AddRecord("192.168.1.2", ipread.MemoryRecord{
		CountryIso: "GB",
		CityName:   "London",
		TimeZone:   "Europe/London",
	})
	
	provider.AddRecord("2001:db8::1", ipread.MemoryRecord{
		CountryIso: "DE",
		CityName:   "Berlin",
		TimeZone:   "Europe/Berlin",
	})
	
	// Test first record
	ip1 := net.ParseIP("192.168.1.1")
	city1, err1 := provider.City(ip1)
	s.NoError(err1)
	s.Equal("US", city1.Country.IsoCode)
	s.Equal("New York", city1.City.Names["en"])
	
	// Test second record
	ip2 := net.ParseIP("192.168.1.2")
	city2, err2 := provider.City(ip2)
	s.NoError(err2)
	s.Equal("GB", city2.Country.IsoCode)
	s.Equal("London", city2.City.Names["en"])
	
	// Test IPv6 record
	ip3 := net.ParseIP("2001:db8::1")
	city3, err3 := provider.City(ip3)
	s.NoError(err3)
	s.Equal("DE", city3.Country.IsoCode)
	s.Equal("Berlin", city3.City.Names["en"])
	
	// Test non-existent record
	ip4 := net.ParseIP("192.168.1.3")
	city4, err4 := provider.City(ip4)
	s.Error(err4)
	s.Nil(city4)
	s.Equal(ipread.ErrIpNotFound, err4)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_AddRecord_Overwrite() {
	provider := ipread.ProvideMemoryProvider("test-overwrite")
	
	// Add initial record
	provider.AddRecord("192.168.1.1", ipread.MemoryRecord{
		CountryIso: "US",
		CityName:   "New York",
		TimeZone:   "America/New_York",
	})
	
	// Overwrite with new record
	provider.AddRecord("192.168.1.1", ipread.MemoryRecord{
		CountryIso: "CA",
		CityName:   "Toronto",
		TimeZone:   "America/Toronto",
	})
	
	ip := net.ParseIP("192.168.1.1")
	city, err := provider.City(ip)
	
	s.NoError(err)
	s.NotNil(city)
	s.Equal("CA", city.Country.IsoCode)
	s.Equal("Toronto", city.City.Names["en"])
	s.Equal("America/Toronto", city.Location.TimeZone)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_AddRecord_EmptyFields() {
	provider := ipread.ProvideMemoryProvider("test-empty")
	
	record := ipread.MemoryRecord{
		CountryIso: "",
		CityName:   "",
		TimeZone:   "",
	}
	
	provider.AddRecord("192.168.1.1", record)
	
	ip := net.ParseIP("192.168.1.1")
	city, err := provider.City(ip)
	
	s.NoError(err)
	s.NotNil(city)
	s.Equal("", city.Country.IsoCode)
	s.Equal("", city.City.Names["en"])
	s.Equal("", city.Location.TimeZone)
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_Refresh() {
	provider := ipread.ProvideMemoryProvider("test-refresh")
	
	err := provider.Refresh(s.ctx)
	
	s.NoError(err, "Refresh should always succeed for memory provider")
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_Close() {
	provider := ipread.ProvideMemoryProvider("test-close")
	
	err := provider.Close()
	
	s.NoError(err, "Close should always succeed for memory provider")
}

func (s *MemoryProviderTestSuite) TestMemoryProvider_Integration_WithReader() {
	// Test integration between memory provider and reader
	s.setupConfigValues(map[string]interface{}{
		"ipread.integration.provider": "memory",
	})
	
	// Set up logger expectation for this test since it calls NewReader
	mockLogger := s.logger.(*logMocks.Logger)
	mockLogger.On("WithChannel", "ipread").Return(mockLogger)
	
	reader, err := ipread.NewReader(s.ctx, s.config, s.logger, "integration")
	s.NoError(err)
	
	// Add a record to the memory provider for this reader
	provider := ipread.ProvideMemoryProvider("integration")
	provider.AddRecord("203.0.113.1", ipread.MemoryRecord{
		CountryIso: "JP",
		CityName:   "Tokyo",
		TimeZone:   "Asia/Tokyo",
	})
	
	// Test the reader
	geoCity, err := reader.City("203.0.113.1")
	s.NoError(err)
	s.NotNil(geoCity)
	s.Equal("203.0.113.1", geoCity.Ip)
	s.Equal("JP", geoCity.CountryCode)
	s.Equal("Tokyo", geoCity.City)
	s.Equal("Asia/Tokyo", geoCity.TimeZone)
	
	// Test with IP not in memory
	geoCity2, err2 := reader.City("203.0.113.2")
	s.Error(err2)
	s.Nil(geoCity2)
}