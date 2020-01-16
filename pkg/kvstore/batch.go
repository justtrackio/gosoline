package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/refl"
)

func keyChunks(keys []interface{}, size int) [][]interface{} {
	var chunks [][]interface{}

	for i := 0; i < len(keys); i += size {
		end := i + size

		if end > len(keys) {
			end = len(keys)
		}

		chunks = append(chunks, keys[i:end])
	}

	return chunks
}

type chunkGetter func(ctx context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error)

func getBatch(ctx context.Context, keys interface{}, result interface{}, getChunk chunkGetter, batchSize int) ([]interface{}, error) {
	missing := make([]interface{}, 0)
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

	for _, chunk := range chunks {
		miss, err := getChunk(ctx, resultMap, chunk)

		if err != nil {
			return keySlice, fmt.Errorf("can not get chunk: %w", err)
		}

		missing = append(missing, miss...)
	}

	return missing, nil
}
