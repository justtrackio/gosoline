package sns

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

func buildAttributes(attributes []map[string]string) (map[string]types.MessageAttributeValue, error) {
	if len(attributes) == 0 {
		return nil, nil
	}

	snsAttributes := map[string]types.MessageAttributeValue{}

	for _, attrs := range attributes {
		for key, val := range attrs {
			if !IsValidAttributeName(key) {
				continue
			}

			snsAttributes[key] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(val),
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

func buildFilterPolicy(attributes map[string]string) (string, error) {
	bytes, err := json.Marshal(attributes)
	if err != nil {
		return "", fmt.Errorf("can not marshal attributes to json: %w", err)
	}

	return string(bytes), nil
}
