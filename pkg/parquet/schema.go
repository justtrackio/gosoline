package parquet

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/refl"
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
	reflect.TypeOf(time.Time{}): "INT96",
}

func mapFieldsToTags(value interface{}) (map[string]interface{}, error) {
	rt, rv := refl.ResolveBaseTypeAndValue(value)

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

			result[fieldName] = createParquetInt96Timestamp(t)
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
	baseType, _ := refl.ResolveBaseTypeAndValue(items)

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

// Note: firehose stores timestamp_millis as nano + julian date little endian int96 values (old parquet format)
// the corresponding go type would be a 12 byte byte array, with
// - the first 8 bytes are the nanoseconds within the day
// - the following four bytes represent the julian date

func createParquetInt96Timestamp(t time.Time) string {
	// convert timestamp to julian date and nanoseconds within the day
	// based on https://github.com/carlosjhr64/jd/blob/master/jd.go

	i := t.Year()
	j := int(t.Month())
	k := t.Day()

	v := k - 32075 + 1461*(i+4800+(j-14)/12)/4 + 367*(j-2-(j-14)/12*12)/12 - 3*((i+4900+(j-14)/12)/100)/4

	hour, minute, second := t.Clock()
	nanos := t.Nanosecond()
	nano := time.Duration(hour)*time.Hour +
		time.Duration(minute)*time.Minute +
		time.Duration(second)*time.Second +
		time.Duration(nanos)

	// xitongsys / parquet requires a big endian int96 string representation to write it the same way as firehose

	parquetDate := make([]byte, 12)

	binary.BigEndian.PutUint32(parquetDate[:4], uint32(v))
	binary.BigEndian.PutUint64(parquetDate[4:], uint64(nano))

	bi := big.NewInt(0)
	bi.SetBytes(parquetDate)

	return bi.String()
}

func parseInt96Timestamp(t string) time.Time {
	parquetDate := []byte(t)

	// split into julian date and nano seconds within the day

	nano := binary.LittleEndian.Uint64(parquetDate[:8])
	dt := binary.LittleEndian.Uint32(parquetDate[8:])

	// julian date to Y-m-d conversion based on https://github.com/carlosjhr64/jd/blob/master/jd.go#L24

	l := dt + 68569
	n := 4 * l / 146097
	l = l - (146097*n+3)/4
	i := 4000 * (l + 1) / 1461001
	l = l - 1461*i/4 + 31
	j := 80 * l / 2447
	k := l - 2447*j/80
	l = j / 11
	j = j + 2 - 12*l
	i = 100*(n-49) + i + l

	tm := time.Date(int(i), time.Month(j), int(k), 0, 0, 0, 0, time.UTC)
	tm = tm.Add(time.Duration(nano))

	return tm
}
