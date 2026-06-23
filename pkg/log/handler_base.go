package log

import (
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

// handlerBase provides shared state and logic for handlers that support config-driven
// per-channel log levels with caching.
type handlerBase struct {
	config   cfg.Config
	lck      sync.RWMutex
	level    int
	channels map[string]*int
	name     string
}

// ChannelLevel returns the specific log level configured for a given channel, or nil if none is set.
func (h *handlerBase) ChannelLevel(name string) (*int, error) {
	h.lck.RLock()
	cached, ok := h.channels[name]
	h.lck.RUnlock()

	if ok {
		return cached, nil
	}

	h.lck.Lock()
	defer h.lck.Unlock()

	key := fmt.Sprintf("%s.channels.%s", getHandlerConfigKey(h.name), name)
	settings := &ChannelSetting{}
	if err := h.config.UnmarshalKey(key, settings); err != nil {
		h.channels[name] = nil

		return nil, fmt.Errorf("can not unmarshal channel settings: %w", err)
	}

	if settings.Level == "" {
		h.channels[name] = nil

		return nil, nil
	}

	priority, ok := LevelPriority(settings.Level)
	if !ok {
		h.channels[name] = nil

		return nil, fmt.Errorf("invalid log level priority %q", settings.Level)
	}

	h.channels[name] = &priority

	return &priority, nil
}

// Level returns the default log level priority for this handler.
func (h *handlerBase) Level() int {
	return h.level
}
