package coffin

import (
	"context"
	"fmt"
	"runtime/pprof"
)

// TODO: remove
func RunWithLabels[R any](f func() R, labels map[string]string) R {
	labelArgs := make([]string, 0, len(labels)*2)
	for k, v := range labels {
		labelArgs = append(labelArgs, k, v)
	}

	labelSet := pprof.Labels(labelArgs...)

	var result R
	pprof.Do(context.Background(), labelSet, func(ctx context.Context) {
		result = f()
	})

	return result
}

// TODO: remove
func Named(name string, args ...any) map[string]string {
	return map[string]string{
		"name": fmt.Sprintf(name, args...),
	}
}
