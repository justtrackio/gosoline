package appctx

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/mapx"
)

func MetadataSet(ctx context.Context, key string, value interface{}) error {
	var err error
	var metadata *Metadata

	if metadata, err = ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access metadata: %w", err)
	}

	metadata.Set(key, value)

	return nil
}

func MetadataAppend(ctx context.Context, key string, values ...interface{}) error {
	var err error
	var metadata *Metadata

	if metadata, err = ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access metadata: %w", err)
	}

	return metadata.Append(key, values...)
}

type metadataAppCtxKey int

func ProvideMetadata(ctx context.Context) (*Metadata, error) {
	return Provide(ctx, metadataAppCtxKey(0), func() (*Metadata, error) {
		return NewMetadata(), nil
	})
}

type Metadata struct {
	values *mapx.MapX
}

func NewMetadata() *Metadata {
	return &Metadata{
		values: mapx.NewMapX(),
	}
}

func (m *Metadata) Append(key string, values ...interface{}) error {
	if !m.values.Has(key) {
		return m.values.Append(key, values...)
	}

	var err error
	var slice []interface{}

	if slice, err = m.values.Get(key).Slice(); err != nil {
		return err
	}

	for _, val := range values {
		found := false

		for _, elem := range slice {
			if elem == val {
				found = true
				break
			}
		}

		if found {
			continue
		}

		slice = append(slice, val)
	}

	m.values.Set(key, slice)

	return nil
}

func (m *Metadata) Get(key string) *mapx.MapXNode {
	return m.values.Get(key)
}

func (m *Metadata) Msi() map[string]interface{} {
	return m.values.Msi()
}

func (m *Metadata) Set(key string, value interface{}) {
	m.values.Set(key, value)
}
