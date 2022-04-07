package ddb

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type metadataFactory struct{}

func NewMetadataFactory() *metadataFactory {
	return &metadataFactory{}
}

func (f *metadataFactory) GetMetadata(settings *Settings) (*Metadata, error) {
	tableName := TableName(settings)
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

	if attributes, err = ReadAttributes(settings.Main.Model); err != nil {
		return nil, err
	}

	for n, a := range attributes {
		allAttributes[n] = a
	}

	for _, li := range settings.Local {
		if attributes, err = ReadAttributes(li.Model); err != nil {
			return nil, err
		}

		for n, a := range attributes {
			allAttributes[n] = a
		}
	}

	for _, gi := range settings.Global {
		if attributes, err = ReadAttributes(gi.Model); err != nil {
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

	if attributes, err = ReadAttributes(model); err != nil {
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

	hashKey = mdl.Box(hashAttribute.AttributeName)
	if rangeAttribute != nil {
		rangeKey = mdl.Box(rangeAttribute.AttributeName)
	}

	if fields, err = MetadataReadFields(model); err != nil {
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

func ReadAttributes(model interface{}) (Attributes, error) {
	t := findBaseType(model)
	attributes := make(Attributes)

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("can't read attributes from model as it is not a struct but instead is %T", model)
	}

	err := readAttributesFromType(t, attributes)

	return attributes, err
}

func readAttributesFromType(t reflect.Type, attributes Attributes) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			err := readAttributesFromType(field.Type, attributes)
			if err != nil {
				return err
			}
		}

		tag, ok := field.Tag.Lookup("ddb")

		if !ok {
			continue
		}

		tag = strings.TrimSpace(tag)

		if len(tag) == 0 {
			return fmt.Errorf("the ddb tag for field %s is empty", field.Name)
		}

		attributeNamePtr, _, err := getAttributeName(field)
		if err != nil {
			return err
		}

		if attributeNamePtr == nil {
			return fmt.Errorf("the json tag for field %s specifies the field should be dropped, but the field is required by ddb", field.Name)
		}

		attributeName := *attributeNamePtr

		attributes[attributeName] = &Attribute{
			FieldName:     field.Name,
			AttributeName: attributeName,
			Tags:          make(map[string]string),
			Type:          getAttributeType(field),
		}

		parts := strings.Split(tag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			kv := strings.Split(part, "=")

			if len(kv) != 2 {
				return fmt.Errorf("the parts of a ddb tag should have the format x=y on field %s", field.Name)
			}

			key := strings.TrimSpace(kv[0])
			key = strings.ToLower(key)
			value := strings.TrimSpace(kv[1])
			value = strings.ToLower(value)

			attributes[attributeName].Tags[key] = value
		}
	}

	return nil
}

func getAttributeName(field reflect.StructField) (*string, bool, error) {
	jsonTag, ok := field.Tag.Lookup("json")

	if !ok {
		return &field.Name, false, nil
	}

	jsonTag = strings.TrimSpace(jsonTag)

	if len(jsonTag) == 0 {
		return nil, false, fmt.Errorf("the json tag for field %s is empty", field.Name)
	}

	if jsonTag == "-" {
		return nil, false, nil
	}

	jsonTag = strings.SplitN(jsonTag, ",", 2)[0]

	if len(jsonTag) == 0 {
		jsonTag = field.Name
	}

	return &jsonTag, true, nil
}

func getAttributeType(field reflect.StructField) types.ScalarAttributeType {
	var attributeType types.ScalarAttributeType

	t := field.Type

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		attributeType = types.ScalarAttributeTypeS
	case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		attributeType = types.ScalarAttributeTypeN
	case reflect.Struct:
		switch t.String() {
		case reflect.TypeOf(time.Time{}).String():
			attributeType = types.ScalarAttributeTypeS
		default:
			panic(fmt.Errorf("type %s is not supported for kind of struct for attributeType", t.String()))
		}
	default:
		panic(fmt.Errorf("unknown attributeType for field of kind %s with type %s", t.Kind().String(), t.String()))
	}

	return attributeType
}

func MetadataReadFields(model interface{}) ([]string, error) {
	t := findBaseType(model)

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("can't read fields from model as it is not a struct but instead is %T", model)
	}

	return metadataReadFieldsForType(t)
}

func metadataReadFieldsForType(t reflect.Type) ([]string, error) {
	fields := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldName, hasJsonTag, err := getAttributeName(field)
		if err != nil {
			return nil, err
		}

		if field.Anonymous && !hasJsonTag {
			embeddedFields, err := metadataReadFieldsForType(field.Type)
			if err != nil {
				return nil, err
			}

			fields = append(fields, embeddedFields...)
			continue
		}

		if fieldName == nil {
			// field was marked as skipped
			continue
		}

		fields = append(fields, *fieldName)
	}

	return fields, nil
}
