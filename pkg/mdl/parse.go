package mdl

import (
	"fmt"
)

// ParseModelId parses a string into a ModelId using the required domain pattern from config.
// This is the only supported way to parse a canonical model id string.
//
// Returns an error if:
//   - the domain pattern is not configured or invalid
//   - the string doesn't match the expected domain pattern structure
func ParseModelId(config ConfigProvider, s string) (ModelId, error) {
	domainPattern, err := readModelIdDomainPattern(config)
	if err != nil {
		return ModelId{}, err
	}

	id, err := modelIdFromStringWithDomainPattern(domainPattern, s)
	if err != nil {
		return ModelId{}, fmt.Errorf("failed to parse model id %q with domain pattern %q: %w", s, domainPattern, err)
	}

	return id, nil
}

// DebugModelIdString returns a debug representation of a ModelId.
// This is safe to use for logging and does not require config.
// It should NOT be used for routing, persistence keys, or message attributes.
func DebugModelIdString(id ModelId) string {
	return fmt.Sprintf("ModelId{Name:%q, Env:%q, App:%q, Tags:%v}", id.Name, id.Env, id.App, id.Tags)
}

func readModelIdDomainPattern(config ConfigProvider) (string, error) {
	domainPattern, err := config.GetString(ConfigKeyModelIdDomainPattern)
	if err != nil {
		return "", fmt.Errorf("%s must be set: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern == "" {
		return "", fmt.Errorf("%s must not be empty", ConfigKeyModelIdDomainPattern)
	}

	if err := validateModelIdDomainPattern(domainPattern); err != nil {
		return "", fmt.Errorf("invalid %s: %w", ConfigKeyModelIdDomainPattern, err)
	}

	return domainPattern, nil
}
