package status_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/log/status"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestModule(t *testing.T) {
	mgr := status.NewManager()
	logger := logMocks.NewLogger(t)
	logger.EXPECT().WithChannel("status").Return(logger).Once()
	m, err := status.NewModule(mgr)(t.Context(), nil, logger)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cfn := coffin.New()

	cfn.GoWithContext(ctx, m.Run)

	mgr.StartWork("test", 3).ReportDone()
	logger.EXPECT().Info("Work item %s: done", "test").Run(func(format string, args ...any) {
		// we can cancel the context as soon as we know that we will be logging stuff
		// if we do this too early, the module might get a choice between returning and printing logs,
		// but at this point we are already printing
		cancel()
	}).Once()

	err = unix.Kill(unix.Getpid(), unix.SIGUSR1)
	assert.NoError(t, err)

	err = cfn.Wait()
	assert.NoError(t, err)
}

/*******************/
// manager example //
/*******************/

type testModule struct {
	logger        log.Logger
	statusManager status.Manager
	data          chan int
}

func NewTestModule(_ context.Context, _ cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &testModule{
		logger:        logger.WithChannel("test"),
		statusManager: status.ProvideManager(),
		data:          make(chan int),
	}, nil
}

func (m *testModule) Run(ctx context.Context) error {
	// declare a new work item with 3 steps
	mainHandle := m.statusManager.StartWork("main", 3)
	cfn := coffin.New()

	// first step: launch the work method and monitor its success
	cfn.Go(m.statusManager.Monitor("work 1", m.Work))
	mainHandle.ReportProgress(1, 0)

	// second step: launch another work method and also monitor its success
	cfn.GoWithContext(ctx, m.statusManager.MonitorWithContext("work 2", m.WorkWithContext))
	mainHandle.ReportProgress(2, 0)

	// last step: launch a method that publishes two messages for the other workers to consume
	publishHandle := m.statusManager.StartWork("publish", 2)
	cfn.Go(publishHandle.Monitor(func() error {
		m.data <- 1
		publishHandle.ReportProgress(1, 0)
		m.data <- 2
		publishHandle.ReportProgress(2, 0)

		return nil
	}))
	mainHandle.ReportDone()

	// print the report by hand. normally the module takes care of this when it receives a SIGUSR1.
	m.statusManager.PrintReport(m.logger)
	// defer it again to get it printed after all go routines finished
	defer m.statusManager.PrintReport(m.logger)

	// wait for all routines to exit again
	return cfn.Wait()
}

// this method simply waits for a published message and never fails afterwards
func (m *testModule) Work() error {
	<-m.data

	return nil
}

// we also wait for a message, but also handle context cancellation. however, this should not happen in this example
func (m *testModule) WorkWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-m.data:
		return nil
	}
}

func TestModuleExample(t *testing.T) {
	exitCodeHandler := application.WithKernelExitHandler(func(code int) {})

	app := application.New(
		exitCodeHandler,
		application.WithModuleFactory("status", status.NewModule(status.ProvideManager())),
		application.WithModuleFactory("main", NewTestModule),
		application.WithConfigMap(map[string]any{
			"env":         "test",
			"app_project": "justtrack",
			"app_family":  "fam",
			"app_group":   "grp",
			"app_name":    "name",
		}),
	)
	app.Run()
}
