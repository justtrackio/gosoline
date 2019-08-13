package apiserver

import (
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

type CustomValidator struct {
	Name      string
	Validator validator.Func
}

func AddCustomValidators(customValidators []CustomValidator) error {
	for _, customValidator := range customValidators {
		v, ok := binding.Validator.Engine().(*validator.Validate)

		if !ok {
			continue
		}

		err := v.RegisterValidation(customValidator.Name, customValidator.Validator)

		if err != nil {
			return err
		}
	}

	return nil
}
