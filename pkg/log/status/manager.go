package status

import (
	"context"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/log"
	"sort"
	"sync"
)

//go:generate mockery --name Manager
type Manager interface {
	// initialize a new work item (e.g. working through the files of a day) with the given number of steps (e.g. days) to do
	StartWork(key string, steps int) WorkItem
	// print the report to the logger
	PrintReport(logger log.Logger)
	// monitor execution of a method and set the correct result status upon return
	Monitor(key string, f func() error) func() error
	// like Monitor, but pass along a context
	MonitorWithContext(key string, f func(ctx context.Context) error) func(ctx context.Context) error
}

//go:generate mockery --name WorkItem
type WorkItem interface {
	// monitor execution of a method and set the correct result status upon return
	Monitor(f func() error) func() error
	// update the progress of a work item with the current step and the progress (in %) of that step (e.g. how many files of that day are done?)
	ReportProgress(step int, progress float64)
	// utility to report 100% progress on all steps
	ReportDone()
	// report that a work item failed and the reason for that
	ReportError(err error)
}

type manager struct {
	lck  sync.Mutex
	work map[string]*workItem
}

type workItemHandle struct {
	key     string
	manager *manager
}

type workItem struct {
	step       int
	totalSteps int
	progress   float64
	err        error
}

var managerContainer = struct {
	sync.Mutex
	instance Manager
}{}

func ProvideManager() Manager {
	managerContainer.Lock()
	defer managerContainer.Unlock()

	if managerContainer.instance != nil {
		return managerContainer.instance
	}

	managerContainer.instance = NewManager()

	return managerContainer.instance
}

func NewManager() Manager {
	return &manager{
		lck:  sync.Mutex{},
		work: make(map[string]*workItem),
	}
}

func (m *manager) StartWork(key string, steps int) WorkItem {
	m.lck.Lock()
	defer m.lck.Unlock()

	m.work[key] = &workItem{
		step:       0,
		totalSteps: steps,
		progress:   0,
		err:        nil,
	}

	return &workItemHandle{
		key:     key,
		manager: m,
	}
}

func (h *workItemHandle) ReportProgress(step int, progress float64) {
	h.manager.lck.Lock()
	defer h.manager.lck.Unlock()

	if item, ok := h.manager.work[h.key]; ok {
		item.step = step
		item.progress = progress
	}
}

func (h *workItemHandle) ReportDone() {
	h.manager.lck.Lock()
	defer h.manager.lck.Unlock()

	if item, ok := h.manager.work[h.key]; ok {
		item.step = item.totalSteps
		item.progress = 100
	}
}

func (h *workItemHandle) ReportError(err error) {
	h.manager.lck.Lock()
	defer h.manager.lck.Unlock()

	if item, ok := h.manager.work[h.key]; ok {
		item.err = err
	}
}

func (h *workItemHandle) Monitor(f func() error) func() error {
	return func() (err error) {
		defer func() {
			panicErr := coffin.ResolveRecovery(recover())

			if panicErr != nil {
				err = panicErr
			}

			if err != nil {
				h.ReportError(err)
			} else {
				h.ReportDone()
			}
		}()

		return f()
	}
}

func (m *manager) PrintReport(logger log.Logger) {
	m.lck.Lock()
	defer m.lck.Unlock()

	keys := make([]string, 0, len(m.work))

	for key := range m.work {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		work := m.work[key]
		if work.err != nil {
			logger.Info("Work item %s: failed with error %s", key, work.err.Error())
		} else if work.step < work.totalSteps || work.progress < 100 {
			logger.Info("Work item %s: step %d / %d (%.2f %%)", key, work.step, work.totalSteps, work.progress)
		} else {
			logger.Info("Work item %s: done", key)
		}
	}
}

func (m *manager) Monitor(key string, f func() error) func() error {
	return m.StartWork(key, 1).Monitor(f)
}

func (m *manager) MonitorWithContext(key string, f func(ctx context.Context) error) func(ctx context.Context) error {
	h := m.StartWork(key, 1)

	return func(ctx context.Context) error {
		return h.Monitor(func() error {
			return f(ctx)
		})()
	}
}
