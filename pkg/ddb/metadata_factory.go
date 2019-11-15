package ddb

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"reflect"
	"strings"
	"time"
)

type metadataFactory struct{}

func NewMetadataFactory() *metadataFactory {
	return &metadataFactory{}
}

func (f *metadataFactory) GetMetadata(settings *Settings) (*Metadata, error) {
	tableName := namingStrategy(settings.ModelId)
	attributes, err := f.getAttributes(settings)

	if err != nil {
		return nil, fmt.Errorf("can not get attributes for table %s: %w", tableName, err)
	}

	ttl, err := f.getTimeToLive(attributes)

	if err != nil {
		return nil, fmt.Errorf("can not get ttl for table %s: %w", tableName, err)
	}

	mainFields, err := f.getFields(settings.Main.Model, tagKey, tagKey)

	if err != nil {
		return nil, fmt.Errorf("can not get fields for main table %s: %w", tableName, err)
	}

	local, err := f.getLocalSecondaryIndices(settings.Local)

	if err != nil {
		return nil, fmt.Errorf("can not get fields for local secondary index on table %s: %w", tableName, err)
	}

	global, err := f.getGlobalSecondaryIndices(settings.Global)

	if err != nil {
		return nil, fmt.Errorf("can not get fields for global secondary index on table %s: %w", tableName, err)
	}

	metadata := &Metadata{
		TableName:  tableName,
		Attributes: attributes,
		TimeToLive: ttl,
		Main: metadataMain{
			metadataFields: mainFields,
			metadataCapacity: metadataCapacity{
				ReadCapacityUnits:  settings.Main.ReadCapacityUnits,
				WriteCapacityUnits: settings.Main.WriteCapacityUnits,
			},
		},
		Local:  local,
		Global: global,
	}

	return metadata, nil
}

func (f *metadataFactory) getAttributes(settings *Settings) (Attributes, error) {
	var err error
	var attributes Attributes

	allAttributes := make(Attributes)

	if attributes, err = f.readAttributes(settings.Main.Model); err != nil {
		return nil, err
	}

	for n, a := range attributes {
		allAttributes[n] = a
	}

	for _, li := range settings.Local {
		if attributes, err = f.readAttributes(li.Model); err != nil {
			return nil, err
		}

		for n, a := range attributes {
			allAttributes[n] = a
		}
	}

	for _, gi := range settings.Global {
		if attributes, err = f.readAttributes(gi.Model); err != nil {
			return nil, err
		}

		for n, a := range attributes {
			allAttributes[n] = a
		}
	}

	return allAttributes, nil
}

func (f *metadataFactory) getFields(model interface{}, hashTag string, rangeTag string) (metadataFields, error) {
	var err error
	var attributes Attributes
	var hashAttribute, rangeAttribute *Attribute
	var hashKey, rangeKey *string
	var fields []string

	if attributes, err = f.readAttributes(model); err != nil {
		return metadataFields{}, err
	}

	if hashAttribute, err = attributes.GetByTag(hashTag, "hash"); err != nil {
		return metadataFields{}, err
	}

	if hashAttribute == nil {
		return metadataFields{}, fmt.Errorf("no hash key defined")
	}

	if rangeAttribute, err = attributes.GetByTag(rangeTag, "range"); err != nil {
		return metadataFields{}, err
	}

	hashKey = mdl.String(hashAttribute.AttributeName)
	if rangeAttribute != nil {
		rangeKey = mdl.String(rangeAttribute.AttributeName)
	}

	if fields, err = metadataReadFields(model); err != nil {
		return metadataFields{}, err
	}

	data := metadataFields{
		Model:    model,
		Fields:   fields,
		HashKey:  hashKey,
		RangeKey: rangeKey,
	}

	return data, nil
}

func (f *metadataFactory) getLocalSecondaryIndices(settings []LocalSettings) (metaLocal, error) {
	local := make(metaLocal)

	for _, ls := range settings {
		localFields, err := f.getFields(ls.Model, tagKey, tagLocal)

		if err != nil {
			return nil, err
		}

		if localFields.RangeKey == nil {
			return nil, fmt.Errorf("no range key defined for local secondary index")
		}

		name := ls.Name
		if len(name) == 0 {
			name = fmt.Sprintf("local-%s", *localFields.RangeKey)
		}

		if _, ok := local[name]; ok {
			return nil, fmt.Errorf("there is already a local secondary index with the name %s", name)
		}

		local[name] = localFields
	}

	return local, nil
}

func (f *metadataFactory) getGlobalSecondaryIndices(settings []GlobalSettings) (metaGlobal, error) {
	global := make(metaGlobal)

	for _, gs := range settings {
		globalFields, err := f.getFields(gs.Model, tagGlobal, tagGlobal)

		if err != nil {
			return nil, err
		}

		name := gs.Name
		if len(name) == 0 {
			name = fmt.Sprintf("global-%s", *globalFields.HashKey)
		}

		if _, ok := global[name]; ok {
			return nil, fmt.Errorf("there is already a global secondary index with the name %s", name)
		}

		global[name] = metadataMain{
			metadataFields: globalFields,
			metadataCapacity: metadataCapacity{
				ReadCapacityUnits:  gs.ReadCapacityUnits,
				WriteCapacityUnits: gs.WriteCapacityUnits,
			},
		}
	}

	return global, nil
}

func (f *metadataFactory) getTimeToLive(attributes Attributes) (metadataTtl, error) {
	data := metadataTtl{
		Enabled: false,
	}
	ttl, err := attributes.GetByTag("ttl", "enabled")

	if err != nil {
		return data, err
	}

	if ttl == nil {
		return data, err
	}

	data.Enabled = true
	data.Field = ttl.AttributeName

	return data, nil
}

func (f *metadataFactory) readAttributes(model interface{}) (Attributes, error) {
	t := findBaseType(model)
	attributes := make(Attributes)

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("can't read attributes from model as it is not a struct but instead is %T", model)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup("ddb")

		if !ok {
			continue
		}

		if len(tag) == 0 {
			return nil, fmt.Errorf("the ddb tag for field %s is empty", field.Name)
		}

		attributeName := f.getAttributeName(field)
		attributeType := f.getAttributeType(field)

		attributes[attributeName] = &Attribute{
			FieldName:     field.Name,
			AttributeName: attributeName,
			Tags:          make(map[string]string),
			Type:          attributeType,
		}

		parts := strings.Split(tag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			kv := strings.Split(part, "=")

			if len(kv) != 2 {
				return nil, fmt.Errorf("the parts of a ddb tag should have the format x=y on field %s", field.Name)
			}

			key := strings.TrimSpace(kv[0])
			key = strings.ToLower(key)
			value := strings.TrimSpace(kv[1])
			value = strings.ToLower(value)

			attributes[attributeName].Tags[key] = value
		}
	}

	return attributes, nil
}

func (f *metadataFactory) getAttributeName(field reflect.StructField) string {
	attributeName := field.Name

	if jsonTag, ok := field.Tag.Lookup("json"); ok {
		attributeName = strings.TrimSpace(jsonTag)
	}

	return attributeName
}

func (f *metadataFactory) getAttributeType(field reflect.StructField) string {
	attributeType := ""

	t := field.Type

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		attributeType = dynamodb.ScalarAttributeTypeS
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Float32, reflect.Float64:
		attributeType = dynamodb.ScalarAttributeTypeN
	case reflect.Struct:
		switch t.String() {
		case reflect.TypeOf(time.Time{}).String():
			attributeType = dynamodb.ScalarAttributeTypeS
		default:
			panic(fmt.Errorf("type %s is not supported for kind of struct for attributeType", t.String()))
		}
	default:
		panic(fmt.Errorf("unknown attributeType for field of kind %s with type %s", t.Kind().String(), t.String()))
	}

	return attributeType
}

func metadataReadFields(model interface{}) ([]string, error) {
	t := findBaseType(model)
	fields := make([]string, 0)

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("can't read fields from model as it is not a struct but instead is %T", model)
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup("json")

		if !ok {
			fields = append(fields, field.Name)
			continue
		}

		if len(tag) == 0 {
			return nil, fmt.Errorf("the json tag for field %s is empty", field.Name)
		}

		tag = strings.TrimSpace(tag)

		if tag == "-" {
			continue
		}

		fields = append(fields, tag)
	}

	return fields, nil
}
