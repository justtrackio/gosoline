package sns

import (
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/spf13/cast"
)

func buildAttributes(attributes []map[string]interface{}) (map[string]*sns.MessageAttributeValue, error) {
	if len(attributes) == 0 {
		return nil, nil
	}

	var snsAttributes = map[string]*sns.MessageAttributeValue{}

	for _, attrs := range attributes {
		for key, val := range attrs {
			switch v := val.(type) {
			case string:
				snsAttributes[key] = &sns.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(v),
				}

			case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64, float32, float64:
				strVal, err := cast.ToStringE(val)

				if err != nil {
					return nil, fmt.Errorf("number %v of key %s is not castable to string", val, key)
				}

				snsAttributes[key] = &sns.MessageAttributeValue{
					DataType:    aws.String("Number"),
					StringValue: aws.String(strVal),
				}

			default:
				return nil, fmt.Errorf("data type %T of key %s is not supported", val, key)
			}
		}
	}

	return snsAttributes, nil
}

func buildFilterPolicy(attributes map[string]interface{}) (string, error) {
	bytes, err := json.Marshal(attributes)

	if err != nil {
		return "", fmt.Errorf("can not marshal attributes to json: %w", err)
	}

	return string(bytes), nil
}
