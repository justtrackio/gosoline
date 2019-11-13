package parquet

import (
	"encoding/binary"
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/refl"
	"reflect"
	"time"
)

const (
	tagName        = "parquet"
	propName       = "name"
	propType       = "type"
	propRepetition = "repetitionType"
)

type tagProps map[string]string

type schemaNode struct {
	Tag    string       `json:"Tag"`
	Fields []schemaNode `json:"Fields,omitempty"`
}

var schemaTypeMap = map[reflect.Type]string{
	reflect.TypeOf(true):        "BOOLEAN",
	reflect.TypeOf(int(0)):      "INT_32",
	reflect.TypeOf(int8(0)):     "INT_8",
	reflect.TypeOf(int16(0)):    "INT_16",
	reflect.TypeOf(int32(0)):    "INT_32",
	reflect.TypeOf(int64(0)):    "INT_64",
	reflect.TypeOf(uint(0)):     "UINT_32",
	reflect.TypeOf(uint8(0)):    "UINT_8",
	reflect.TypeOf(uint16(0)):   "UINT_16",
	reflect.TypeOf(uint32(0)):   "UINT_32",
	reflect.TypeOf(uint64(0)):   "UINT_64",
	reflect.TypeOf(float32(0)):  "FLOAT",
	reflect.TypeOf(float64(0)):  "DOUBLE",
	reflect.TypeOf(""):          "UTF8",
	reflect.TypeOf([]byte{}):    "BYTE_ARRAY",
	reflect.TypeOf(time.Time{}): "TIMESTAMP_MILLIS",
}

func mapFieldsToTags(value interface{}) (map[string]interface{}, error) {
	rt := reflect.TypeOf(value)
	rv := reflect.ValueOf(value)

	if rt.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Cannot map %T to struct tags", value))
	}

	result := make(map[string]interface{}, rt.NumField())

	for i := 0; i < rt.NumField(); i++ {
		tField := rt.Field(i)
		vField := rv.Field(i)

		fieldName, ok := tField.Tag.Lookup(tagName)

		if !ok {
			continue
		}

		if vField.Kind() == reflect.Ptr {
			vField = vField.Elem()
		}

		if !vField.IsValid() {
			continue
		}

		v := vField.Interface()

		switch reflect.TypeOf(v) {
		case reflect.TypeOf(time.Time{}):
			t, ok := v.(time.Time)

			if !ok {
				return nil, fmt.Errorf("could not access time field %s for conversion: type %T", tField.Name, vField.Interface())
			}

			result[fieldName] = createTimestamp(t)
		default:
			result[fieldName] = v
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("could not find any tags named %s at type %T", tagName, value)
	}

	return result, nil
}

func parseSchema(items interface{}) (string, error) {
	baseType := refl.FindBaseType(items)

	rootTag := createTag(tagProps{
		propName:       "parquet-go-root",
		propRepetition: "REQUIRED",
	})

	rootNode := createNode(rootTag, baseType)

	marshalled, err := json.Marshal(rootNode)

	return string(marshalled), err
}

func createNode(schemaTag string, prop reflect.Type) schemaNode {
	node := schemaNode{
		Tag:    schemaTag,
		Fields: make([]schemaNode, 0),
	}

	if prop.Kind() != reflect.Struct {
		return node
	}

	for i := 0; i < prop.NumField(); i++ {
		field := prop.Field(i)

		fieldName, ok := field.Tag.Lookup(tagName)

		if !ok {
			continue
		}

		t := field.Type

		repetition := "REQUIRED"

		if t.Kind() == reflect.Ptr {
			repetition = "OPTIONAL"
			t = t.Elem()
		}

		fieldType, ok := schemaTypeMap[t]

		if !ok {
			continue
		}

		propSchemaTag := createTag(tagProps{
			propName:       fieldName,
			propType:       fieldType,
			propRepetition: repetition,
		})

		propNode := createNode(propSchemaTag, prop.Field(i).Type)
		node.Fields = append(node.Fields, propNode)
	}

	return node
}

func createTag(props map[string]string) string {
	tag := ""

	for key, value := range props {
		if len(tag) > 0 {
			tag += ", "
		}

		tag += fmt.Sprintf("%s=%s", key, value)
	}

	return tag
}

func createTimestamp(t time.Time) string {
	// based on https://github.com/carlosjhr64/jd/blob/master/jd.go

	year := t.Year()
	month := int(t.Month())
	day := t.Day()

	v := day - 32075 + 1461*(year+4800+(month-14)/12)/4 + 367*(month-2-(month-14)/12*12)/12 - 3*((year+4900+(month-14)/12)/100)/4

	hour, minute, second := t.Clock()
	nano := (hour*3600 + minute*60 + second) * 1000

	parquetDate := make([]byte, 12)

	binary.LittleEndian.PutUint64(parquetDate[:8], uint64(nano))
	binary.LittleEndian.PutUint32(parquetDate[8:], uint32(v))

	return string(parquetDate)
}
