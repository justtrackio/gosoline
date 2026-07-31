package kernel_test

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type loggerTestModule struct {
	cancel context.CancelFunc
}

func (m loggerTestModule) Run(context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

func TestLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}

type LoggerTestSuite struct {
	suite.Suite

	ctx          context.Context
	config       *cfgMocks.Config
	logger       *logMocks.GosoLogger
	kernelLogger *logMocks.Logger
	exitCode     int
	cancel       context.CancelFunc
}

func (s *LoggerTestSuite) SetupTest() {
	s.ctx = appctx.WithContainer(s.T().Context())
	s.config = cfgMocks.NewConfig(s.T())
	s.config.EXPECT().UnmarshalKey("kernel", mock.AnythingOfType("*kernel.Settings")).
		Run(func(_ string, value any, _ ...cfg.UnmarshalDefaults) {
			settings := value.(*kernel.Settings)
			settings.KillTimeout = time.Second
			settings.HealthCheck.Timeout = time.Second
			settings.HealthCheck.WaitInterval = time.Second
		}).
		Return(nil).
		Twice()

	s.logger = logMocks.NewGosoLogger(s.T())
	s.kernelLogger = logMocks.NewLogger(s.T())
	s.logger.EXPECT().WithChannel("kernel").Return(s.kernelLogger).Twice()
	s.kernelLogger.EXPECT().Info(matcher.Context, "starting kernel").Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "running %s module %s in stage %d", "foreground", "module", kernel.StageApplication).Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "stage %d up and running with %d modules", kernel.StageApplication, 1).Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "kernel up and running after %s", mock.AnythingOfType("time.Duration")).Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "stopping kernel due to: %s", mock.AnythingOfType("string")).Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "stopping stage %d", kernel.StageApplication).Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "stopped %s module %s", "foreground", "module").Once()
	s.kernelLogger.EXPECT().Info(matcher.Context, "stopped stage %d", kernel.StageApplication).Maybe()
	s.kernelLogger.EXPECT().Info(matcher.Context, "leaving kernel with exit code %d", kernel.ExitCodeOk).Once()
	s.exitCode = kernel.ExitCodeErr
}

func (s *LoggerTestSuite) buildKernel() kernel.Kernel {
	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("module", func(context.Context, cfg.Config, log.Logger) (kernel.Module, error) {
			return loggerTestModule{cancel: s.cancel}, nil
		}),
		kernel.WithExitHandler(func(code int) {
			s.exitCode = code
		}),
	})
	require.NoError(s.T(), err)

	return k
}

// TestRunsWithDerivedLoggerWithoutClose verifies the kernel closes the root logger without closing its derived channel logger.
func (s *LoggerTestSuite) TestRunsWithDerivedLoggerWithoutClose() {
	s.logger.EXPECT().Close(matcher.Context).Return(nil).Once()

	assert.NotPanics(s.T(), func() {
		s.buildKernel().Run()
	})

	assert.Equal(s.T(), kernel.ExitCodeOk, s.exitCode)
}

// TestExitReportsLoggerCloseError verifies a root logger close failure is reported to stdout and changes the exit code.
func (s *LoggerTestSuite) TestExitReportsLoggerCloseError() {
	s.logger.EXPECT().Close(matcher.Context).Return(errors.New("logger close failed")).Once()

	output := captureStdout(s.T(), s.buildKernel().Run)

	assert.Equal(s.T(), kernel.ExitCodeErr, s.exitCode)
	assert.Equal(s.T(), "close logger completed with errors: logger close failed\n", output)
}

func (s *LoggerTestSuite) TestCloseUsesBoundedLiveAppContext() {
	type contextKey struct{}
	s.ctx, s.cancel = context.WithCancel(context.WithValue(s.ctx, contextKey{}, "value"))
	s.logger.EXPECT().Close(matcher.Context).Run(func(ctx context.Context) {
		assert.NoError(s.T(), ctx.Err())
		assert.Equal(s.T(), "value", ctx.Value(contextKey{}))

		deadline, ok := ctx.Deadline()
		assert.True(s.T(), ok)
		assert.Positive(s.T(), time.Until(deadline))
		assert.LessOrEqual(s.T(), time.Until(deadline), time.Second)
	}).Return(nil).Once()

	s.buildKernel().Run()

	assert.Equal(s.T(), kernel.ExitCodeOk, s.exitCode)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	stdout := os.Stdout
	os.Stdout = writer
	defer func() {
		os.Stdout = stdout
	}()

	fn()

	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return string(output)
}
