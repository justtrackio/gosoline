package apiserver

import (
	"context"
	"fmt"

	"github.com/go-playground/mold/v4/modifiers"
)

var defaultModifier = newDefaultModifier()

type Modifier interface {
	Struct(ctx context.Context, v interface{}) error
}

func WithCustomModifier(modifier Modifier) {
	defaultModifier = modifier
}

func newDefaultModifier() Modifier {
	mod := modifiers.New()
	mod.SetTagName("mold")

	return mod
}

func modifyInput(ctx context.Context, input interface{}) error {
	err := defaultModifier.Struct(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to modify input: %w", err)
	}

	return err
}
