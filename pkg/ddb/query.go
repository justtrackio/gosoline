package ddb

import "github.com/adjoeio/djoemo"

const (
	Equal          djoemo.Operator = "EQ"
	NotEqual                       = "NE"
	Less                           = "LT"
	LessOrEqual                    = "LE"
	Greater                        = "GT"
	GreaterOrEqual                 = "GE"
	BeginsWith                     = "BEGINS_WITH"
	Between                        = "BETWEEN"
)

type query struct {
	tableName    string
	hashKeyName  string
	hashKey      interface{}
	rangeKeyName string
	rangeOp      djoemo.Operator
	rangeKey     interface{}
}

// TableName returns the djoemo table name
func (k *query) TableName() string {
	return k.tableName
}

// HashKeyName returns the name of hash key if exists
func (k *query) HashKeyName() *string {
	return &k.hashKeyName
}

// HashKey returns the hash key value
func (k *query) HashKey() interface{} {
	return k.hashKey
}

// RangeKeyName returns the name of range key if exists
func (k *query) RangeKeyName() *string {
	return &k.rangeKeyName
}

func (k *query) RangeOp() djoemo.Operator {
	return k.rangeOp
}

// HashKey returns the range key value
func (k *query) RangeKey() interface{} {
	return k.rangeKey
}
