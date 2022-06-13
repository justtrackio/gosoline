package sns

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/spf13/cast"
)

func buildAttributes(attributes []map[string]interface{}) (map[string]types.MessageAttributeValue, error) {
	if len(attributes) == 0 {
		return nil, nil
	}

	snsAttributes := map[string]types.MessageAttributeValue{}

	for _, attrs := range attributes {
		for key, val := range attrs {
			if !IsValidAttributeName(key) {
				continue
			}

			switch v := val.(type) {
			case string:
				snsAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(v),
				}

			case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64, float32, float64:
				strVal, err := cast.ToStringE(val)
				if err != nil {
					return nil, fmt.Errorf("number %v of key %s is not castable to string", val, key)
				}

				snsAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("Number"),
					StringValue: aws.String(strVal),
				}

			case bool:
				snsAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(strconv.FormatBool(v)),
				}

			case fmt.Stringer:
				snsAttributes[key] = types.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(v.String()),
				}

			default:
				return nil, fmt.Errorf("data type %T of key %s is not supported", val, key)
			}
		}
	}

	return snsAttributes, nil
}

var validAttributeRegex = regexp.MustCompile(`^[a-z\d_\-]+(\.[a-z\d_\-]+)*$`)

func IsValidAttributeName(name string) bool {
	// https://docs.aws.amazon.com/sns/latest/dg/sns-message-attributes.html#SNSMessageAttributes.DataTypes:
	//
	// The message attribute name can contain the following characters: A-Z, a-z, 0-9, underscore(_), hyphen(-), and period (.).
	// The name must not start or end with a period, and it should not have successive periods. The name is case-sensitive
	// and must be unique among all attribute names for the message. The name can be up to 256 characters long. The name
	// cannot start with "AWS." or "Amazon." (or any variations in casing) because these prefixes are reserved for use
	// by Amazon Web Services.

	name = strings.ToLower(name)

	if strings.HasPrefix(name, "aws.") || strings.HasPrefix(name, "amazon.") {
		return false
	}

	if len(name) > 256 {
		return false
	}

	return validAttributeRegex.MatchString(name)
}

func buildFilterPolicy(attributes map[string]interface{}) (string, error) {
	bytes, err := json.Marshal(attributes)
	if err != nil {
		return "", fmt.Errorf("can not marshal attributes to json: %w", err)
	}

	return string(bytes), nil
}
