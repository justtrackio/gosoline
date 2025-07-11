package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/refl"
)

func keyChunks(keys []any, size int) [][]any {
	var chunks [][]any

	for i := 0; i < len(keys); i += size {
		end := i + size

		if end > len(keys) {
			end = len(keys)
		}

		chunks = append(chunks, keys[i:end])
	}

	return chunks
}

type chunkGetter func(ctx context.Context, resultMap *refl.Map, keys []any) ([]any, error)

func getBatch(ctx context.Context, keys any, result any, getChunk chunkGetter, batchSize int) ([]any, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	keySlice, err = UniqKeys(keySlice)
	if err != nil {
		return nil, fmt.Errorf("can not deduplicate keys: %w", err)
	}

	resultMap, err := refl.MapOf(result)
	if err != nil {
		return keySlice, fmt.Errorf("can not use provided result value: %w", err)
	}

	if batchSize < 1 {
		batchSize = 1
	}

	chunks := keyChunks(keySlice, batchSize)
	missing := make([]any, 0)

	for _, chunk := range chunks {
		miss, err := getChunk(ctx, resultMap, chunk)
		if err != nil {
			return keySlice, fmt.Errorf("can not get chunk: %w", err)
		}

		missing = append(missing, miss...)
	}

	return missing, nil
}
