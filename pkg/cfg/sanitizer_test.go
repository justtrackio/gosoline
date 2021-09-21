package cfg_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/objx"
	"github.com/stretchr/testify/assert"
)

func TestTimeSanitizer(t *testing.T) {
	tm := time.Date(2019, time.November, 26, 0, 0, 0, 0, time.UTC)
	san, err := cfg.TimeSanitizer(tm)

	assert.NoError(t, err)
	assert.Equal(t, "2019-11-26T00:00:00Z", san)

	i := 1337
	san, err = cfg.TimeSanitizer(i)

	assert.NoError(t, err)
	assert.Equal(t, 1337, san)
}

func TestSanitize(t *testing.T) {
	in := map[string]interface{}{
		"foo":  "bar",
		"date": time.Date(2019, time.November, 26, 0, 0, 0, 0, time.UTC),
		"nested": map[string]interface{}{
			"anotherDate": time.Date(2019, time.November, 26, 0, 0, 0, 0, time.UTC),
		},
	}

	san, err := cfg.Sanitize("root", in, []cfg.Sanitizer{
		cfg.TimeSanitizer,
	})

	assert.NoError(t, err)
	assert.IsType(t, map[string]interface{}{}, san)

	s := objx.New(san)

	assert.Equal(t, "bar", s.Get("foo").Data())
	assert.Equal(t, "2019-11-26T00:00:00Z", s.Get("date").Data())
	assert.Equal(t, "2019-11-26T00:00:00Z", s.Get("nested.anotherDate").Data())
}
