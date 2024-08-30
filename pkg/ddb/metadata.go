package ddb

import "slices"

const (
	tagKey    = "key"
	tagLocal  = "local"
	tagGlobal = "global"
)

type KeyAware interface {
	IsKeyField(field string) bool
	GetHashKey() *string
	GetRangeKey() *string
	GetKeyFields() []string
}

type FieldAware interface {
	KeyAware
	GetModel() interface{}
	ContainsField(field string) bool
	GetFields() []string
}

type Metadata struct {
	TableName  string
	Attributes Attributes
	TimeToLive metadataTtl
	Main       metadataMain
	Local      metaLocal
	Global     metaGlobal
}

func (d *Metadata) Index(name string) FieldAware {
	for n, m := range d.Local {
		if n == name {
			return m
		}
	}

	for n, m := range d.Global {
		if n == name {
			return m
		}
	}

	return nil
}

type metadataTtl struct {
	Enabled bool
	Field   string
}

type metadataFields struct {
	Model    interface{}
	Fields   []string
	HashKey  *string
	RangeKey *string
}

func (f metadataFields) GetModel() interface{} {
	return f.Model
}

func (f metadataFields) GetHashKey() *string {
	return f.HashKey
}

func (f metadataFields) GetRangeKey() *string {
	return f.RangeKey
}

func (f metadataFields) IsKeyField(field string) bool {
	if f.HashKey != nil && *f.HashKey == field {
		return true
	}

	if f.RangeKey != nil && *f.RangeKey == field {
		return true
	}

	return false
}

func (f metadataFields) GetKeyFields() []string {
	fields := make([]string, 0)

	if f.HashKey != nil {
		fields = append(fields, *f.HashKey)
	}

	if f.RangeKey != nil {
		fields = append(fields, *f.RangeKey)
	}

	return fields
}

func (f metadataFields) ContainsField(field string) bool {
	return slices.Contains(f.Fields, field)
}

func (f metadataFields) GetFields() []string {
	return f.Fields
}

type metadataCapacity struct {
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type metadataMain struct {
	metadataFields
	metadataCapacity
}

type (
	metaLocal  map[string]metadataFields
	metaGlobal map[string]metadataMain
)
