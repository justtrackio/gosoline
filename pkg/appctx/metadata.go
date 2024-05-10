package appctx

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mapx"
)

// MetadataSet sets the value for key to the provided value, overwriting any existing value.
// The metadata carrier comes from ctx.
func MetadataSet(ctx context.Context, key string, value any) error {
	var err error
	var metadata *Metadata

	if metadata, err = ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access metadata: %w", err)
	}

	metadata.Set(key, value)

	return nil
}

// MetadataAppend appends the provided values to any existing values at key.
// The metadata carrier comes from ctx.
func MetadataAppend(ctx context.Context, key string, values ...any) error {
	var err error
	var metadata *Metadata

	if metadata, err = ProvideMetadata(ctx); err != nil {
		return fmt.Errorf("can not access metadata: %w", err)
	}

	return metadata.Append(key, values...)
}

type metadataAppCtxKey int

// ProvideMetadata retrieves the metadata carrier from ctx if one is present, else it creates a new one.
// This is done through [Provide], thus having the metadata globally available.
func ProvideMetadata(ctx context.Context) (*Metadata, error) {
	return Provide(ctx, metadataAppCtxKey(0), func() (*Metadata, error) {
		return NewMetadata(), nil
	})
}

// Metadata provides a thread safe key value store intended to be used with the container injected by [WithContainer].
type Metadata struct {
	values *mapx.MapX
}

// NewMetadata creates a new Metadata key value store
func NewMetadata() *Metadata {
	return &Metadata{
		values: mapx.NewMapX(),
	}
}

// Append appends the provided values to any values present at key. If no values are present, they are set to values.
// If there are already values present no duplicates will be added (the comparison is performed through [reflect.DeepEqual]).
func (m *Metadata) Append(key string, values ...any) error {
	if !m.values.Has(key) {
		return m.values.Append(key, values...)
	}

	var err error
	var slice []any

	if slice, err = m.values.Get(key).Slice(); err != nil {
		return err
	}

	for _, val := range values {
		if funk.Contains(slice, val) {
			continue
		}

		slice = append(slice, val)
	}

	m.values.Set(key, slice)

	return nil
}

// Get retrieves the value node for key.
// If the key does not exist the data in the node will be empty.
func (m *Metadata) Get(key string) *mapx.MapXNode {
	return m.values.Get(key)
}

// Msi returns a map of all keys to values.
// Implements the [mapx.Msier] interface.
func (m *Metadata) Msi() map[string]any {
	return m.values.Msi()
}

// Set sets the values at key to the provided values, overwriting any already present values.
func (m *Metadata) Set(key string, value any) {
	m.values.Set(key, value)
}
