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
	var err error
	var domainPattern string
	var id ModelId

	if domainPattern, err = config.GetString(ConfigKeyModelIdDomainPattern); err != nil {
		return ModelId{}, fmt.Errorf("%s must be set: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern == "" {
		return ModelId{}, fmt.Errorf("%s must not be empty", ConfigKeyModelIdDomainPattern)
	}

	if err = validateModelIdDomainPattern(domainPattern); err != nil {
		return ModelId{}, fmt.Errorf("invalid %s: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if id, err = modelIdFromStringWithDomainPattern(domainPattern, s); err != nil {
		return ModelId{}, fmt.Errorf("failed to parse model id %q with domain pattern %q: %w", s, domainPattern, err)
	}

	return id, nil
}
