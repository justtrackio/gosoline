package mdlsub

import (
	"errors"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/stream"
)

// UnknownModelError is returned when a model id is not found in the transformer list.
type UnknownModelError struct {
	ModelId string
}

func (e UnknownModelError) Error() string {
	return fmt.Sprintf("unknown model: there is no transformer for modelId %s", e.ModelId)
}

func (e UnknownModelError) As(target any) bool {
	if t, ok := target.(*UnknownModelError); ok {
		*t = e

		return true
	}

	return false
}

// IsIgnorableWithSettings implements stream.IgnorableGetModelError.
func (e UnknownModelError) IsIgnorableWithSettings(settings stream.IgnoreOnGetModelErrorSettings) bool {
	return settings.UnknownModel
}

func NewUnknownModelError(modelId string) UnknownModelError {
	return UnknownModelError{
		ModelId: modelId,
	}
}

func IsUnknownModelError(err error) bool {
	return errors.As(err, &UnknownModelError{})
}

// UnknownModelVersionError is returned when a version is not found for a model id in the transformer list.
type UnknownModelVersionError struct {
	ModelId string
	Version int
}

func (e UnknownModelVersionError) Error() string {
	return fmt.Sprintf("unknown model version: there is no transformer for modelId %s and version %d", e.ModelId, e.Version)
}

func (e UnknownModelVersionError) As(target any) bool {
	if t, ok := target.(*UnknownModelVersionError); ok {
		*t = e

		return true
	}

	return false
}

// IsIgnorableWithSettings implements stream.IgnorableGetModelError.
func (e UnknownModelVersionError) IsIgnorableWithSettings(settings stream.IgnoreOnGetModelErrorSettings) bool {
	return settings.UnknownVersion
}

func NewUnknownModelVersionError(modelId string, version int) UnknownModelVersionError {
	return UnknownModelVersionError{
		ModelId: modelId,
		Version: version,
	}
}

func IsUnknownModelVersionError(err error) bool {
	return errors.As(err, &UnknownModelVersionError{})
}
