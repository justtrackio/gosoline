package apiserver

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	Name      string
	Validator validator.Func
}

func AddCustomValidators(customValidators []CustomValidator) error {
	for _, customValidator := range customValidators {
		v, ok := binding.Validator.Engine().(*validator.Validate)

		if !ok {
			return fmt.Errorf("invalid validator engine type, expected %T, got %T", &validator.Validate{}, binding.Validator.Engine())
		}

		err := v.RegisterValidation(customValidator.Name, customValidator.Validator)

		if err != nil {
			return err
		}
	}

	return nil
}
