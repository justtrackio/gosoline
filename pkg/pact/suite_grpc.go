package pact

import (
	"fmt"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/pact/matchers"
	message "github.com/pact-foundation/pact-go/v2/message/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

type grpcPluginConfig struct {
	AdditionalIncludes []string `json:"additionalIncludes"`
}

type grpcInteraction struct {
	Proto        string           `json:"pact:proto"`
	ProtoService string           `json:"pact:proto-service"`
	ContentType  string           `json:"pact:content-type"`
	ProtoConfig  grpcPluginConfig `json:"pact:protobuf-config"`
	Request      matchers.Matcher `json:"request"`
	Response     matchers.Matcher `json:"response"`
}

type GrpcTestCase struct {
	Run      func(conn *grpc.ClientConn) error
	Request  matchers.Matcher
	Response matchers.Matcher
}

type GrpcConfig struct {
	Provider, Consumer, ProtoFile, ProtoPath, PactDir string
}

type WithSetup interface {
	SetupTest()
}

type WithTeardown interface {
	TeardownTest(t *testing.T)
}

type GrpcTest map[string]map[string]GrpcTestCase

type GrpcSuite interface {
	TestConfig() GrpcConfig
	TestCases() GrpcTest
}

func GenerateGrpcPact(t *testing.T, s GrpcSuite) {
	testConfig := s.TestConfig()
	if withSetup, ok := s.(WithSetup); ok {
		withSetup.SetupTest()
	}

	for service, test := range s.TestCases() {
		for name, test := range test {
			mockProvider, err := message.NewSynchronousPact(message.Config{
				Consumer: testConfig.Consumer,
				Provider: testConfig.Provider,
				PactDir:  testConfig.PactDir,
			})
			if err != nil {
				t.Logf("failed to build pact provider: %s", err.Error())
				t.FailNow()
			}

			if withSetup, ok := s.(WithSetup); ok {
				withSetup.SetupTest()
			}

			interaction := grpcInteraction{
				Proto:        testConfig.ProtoFile,
				ProtoService: service,
				ContentType:  "application/protobuf",
				ProtoConfig: grpcPluginConfig{
					AdditionalIncludes: []string{
						testConfig.ProtoPath,
					},
				},
				Request:  test.Request,
				Response: test.Response,
			}

			interactionEncoded, err := json.MarshalIndent(interaction, "", "\t")
			if err != nil {
				t.Logf("failed to build grpc interactoin: %s", err.Error())
				t.FailNow()
			}

			if err := mockProvider.AddSynchronousMessage(name).
				Given(name).
				UsingPlugin(message.PluginConfig{
					Plugin:  "protobuf",
					Version: "0.3.0",
				}).
				WithContents(string(interactionEncoded), "application/grpc").
				StartTransport("grpc", "127.0.0.1", nil).
				ExecuteTest(t, func(tc message.TransportConfig, m message.SynchronousMessage) (err error) {
					conn, err := grpc.Dial(fmt.Sprintf("%s:%d", tc.Address, tc.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						return err
					}

					return test.Run(conn)
				}); err != nil {
				t.Logf("failed to generate pact: %s", err.Error())
				t.FailNow()
			}

			if withTeardown, ok := s.(WithTeardown); ok {
				withTeardown.TeardownTest(t)
			}
		}
	}
}
