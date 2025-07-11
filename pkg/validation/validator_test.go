package validation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
)

type Model struct {
	Name string
}

type NameRule struct{}

func (NameRule) IsValid(ctx context.Context, model any) error {
	m := model.(*Model)

	if len(m.Name) > 3 {
		return nil
	}

	return errors.New("name not long enough")
}

func TestValidator_IsValidDefaultGroup(t *testing.T) {
	v := buildValidator()
	v.AddRule(&NameRule{})

	m1 := &Model{
		Name: "foobar",
	}

	err := v.IsValid(t.Context(), m1)
	assert.NoError(t, err, "model should be valid")

	m2 := &Model{
		Name: "foo",
	}

	err = v.IsValid(t.Context(), m2)
	assert.Error(t, err, "model should be invalid")
}

func TestValidator_IsValidGroups(t *testing.T) {
	v := buildValidator()
	v.AddRule(&NameRule{}, "bla")

	m2 := &Model{
		Name: "foo",
	}

	err := v.IsValid(t.Context(), m2)
	assert.NoError(t, err, "model should be invalid")

	err = v.IsValid(t.Context(), m2, "bla")
	assert.Error(t, err, "model should be invalid")
}

func buildValidator() validation.Validator {
	tracer := tracing.NewLocalTracer()
	v := validation.NewValidatorWithInterfaces(tracer)

	return v
}
