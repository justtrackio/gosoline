package httpserver

import (
	"fmt"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// A CustomValidator allows you to validate single fields with custom rules. If you need to validate a whole struct,
// you need to use a StructValidator. See https://github.com/go-playground/validator/issues/470 for more details.
type CustomValidator struct {
	Name      string
	Validator validator.Func
}

// A StructValidator validates every instance of a struct type with the given validation. It is a little bit unfortunate
// because you now can't have different validation rules for a struct at different places.
type StructValidator struct {
	Struct    any
	Validator validator.StructLevelFunc
}

// A CustomTypeFunc allows you to convert one type to another type before validation. This allows you to validate custom
// types with the built-in functions for, e.g., integers or strings.
type CustomTypeFunc struct {
	Func  validator.CustomTypeFunc
	Types []any
}

// A ValidateAlias allows you to map one or more tags to a different name, making your validation rules easier to read.
type ValidateAlias struct {
	Alias string
	Tags  string
}

func AddCustomValidators(customValidators []CustomValidator) error {
	v, err := getValidateEngine()
	if err != nil {
		return err
	}

	for _, customValidator := range customValidators {
		err = v.RegisterValidation(customValidator.Name, customValidator.Validator)
		if err != nil {
			return err
		}
	}

	return nil
}

func AddStructValidators(structValidators []StructValidator) error {
	v, err := getValidateEngine()
	if err != nil {
		return err
	}

	for _, structValidator := range structValidators {
		v.RegisterStructValidation(structValidator.Validator, structValidator.Struct)
	}

	return nil
}

func AddCustomTypeFuncs(customTypeFuncs []CustomTypeFunc) error {
	v, err := getValidateEngine()
	if err != nil {
		return err
	}

	for _, customTypeFunc := range customTypeFuncs {
		v.RegisterCustomTypeFunc(customTypeFunc.Func, customTypeFunc.Types...)
	}

	return nil
}

func AddValidateAlias(aliases []ValidateAlias) error {
	v, err := getValidateEngine()
	if err != nil {
		return err
	}

	for _, alias := range aliases {
		v.RegisterAlias(alias.Alias, alias.Tags)
	}

	return nil
}

func getValidateEngine() (*validator.Validate, error) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		return v, nil
	}

	return nil, fmt.Errorf("invalid validator engine type, expected %T, got %T", &validator.Validate{}, binding.Validator.Engine())
}
