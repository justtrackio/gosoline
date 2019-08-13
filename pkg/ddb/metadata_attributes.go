package ddb

import "fmt"

type Attribute struct {
	FieldName     string
	AttributeName string
	Tags          map[string]string
	Type          string
}

func (d *Attribute) HasTag(key string, value string) bool {
	for k, v := range d.Tags {
		if k == key && v == value {
			return true
		}
	}

	return false
}

type Attributes map[string]*Attribute

func (a Attributes) GetByTag(key string, value string) (*Attribute, error) {
	var data *Attribute

	for _, d := range a {
		if d.HasTag(key, value) && data != nil {
			return nil, fmt.Errorf("multiple attributes with same tag %s=%s", key, value)
		}

		if d.HasTag(key, value) {
			data = d
		}
	}

	return data, nil
}
