package stream

import (
	"fmt"
	"math"
)

type (
	Chunk  [][]byte
	Chunks []Chunk
)

func BuildChunks(batch []WritableMessage, size int) (Chunks, error) {
	numOfChunks := int(math.Ceil(float64(len(batch)) / float64(size)))

	errors := make([]error, 0)
	chunks := make(Chunks, numOfChunks)

	for i := 0; i < numOfChunks; i++ {
		chunks[i] = make(Chunk, 0, size)
	}

	j := 0
	for i := 0; i < len(batch); i++ {
		bytes, err := batch[i].MarshalToBytes()
		if err != nil {
			errors = append(errors, err)

			continue
		}

		chunks[j] = append(chunks[j], bytes)

		if len(chunks[j]) == size {
			j++
		}
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("there were %v errors on chunking and marshalling the messages", len(errors))
	}

	return chunks, err
}

func ByteChunkToStrings(chunk Chunk) []string {
	strings := make([]string, len(chunk))

	for i := 0; i < len(chunk); i++ {
		strings[i] = string(chunk[i])
	}

	return strings
}

func ByteChunkToInterfaces(chunk Chunk) []any {
	interfaces := make([]any, len(chunk))

	for i := 0; i < len(chunk); i++ {
		interfaces[i] = chunk[i]
	}

	return interfaces
}
