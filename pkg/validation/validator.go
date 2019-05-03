package validation

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/tracing"
	"strings"
)

const (
	GroupDefault = "default"
)

type Rule interface {
	IsValid(ctx context.Context, model interface{}) error
}

type Group map[string][]Rule

type Validator interface {
	AddRule(rule Rule, groups ...string)
	IsValid(ctx context.Context, model interface{}, groups ...string) error
}

type validator struct {
	tracer tracing.Tracer
	rules  Group
}

func NewValidator(config cfg.Config) *validator {
	tracer := tracing.NewAwsTracer(config)

	return NewValidatorWithInterfaces(tracer)
}

func NewValidatorWithInterfaces(tracer tracing.Tracer) *validator {
	return &validator{
		tracer: tracer,
		rules:  make(Group),
	}
}

func (v *validator) AddRule(rule Rule, groups ...string) {
	if len(groups) == 0 {
		groups = append(groups, GroupDefault)
	}

	for _, g := range groups {
		if _, ok := v.rules[g]; !ok {
			v.rules[g] = make([]Rule, 0)
		}

		v.rules[g] = append(v.rules[g], rule)
	}
}

func (v validator) IsValid(ctx context.Context, model interface{}, groups ...string) error {
	ctx, span := v.tracer.StartSubSpan(ctx, "validator")
	defer span.Finish()

	if len(groups) == 0 {
		groups = append(groups, GroupDefault)
	}

	errs := make([]error, 0)

	for _, g := range groups {
		groupErrs := v.validateGroup(ctx, model, g)
		errs = append(errs, groupErrs...)
	}

	if len(errs) == 0 {
		return nil
	}

	messages := make([]string, len(errs))
	for i := 0; i < len(errs); i++ {
		messages[i] = errs[i].Error()
	}

	msg := fmt.Sprintf("validation: %s", strings.Join(messages, "; "))

	return errors.New(msg)
}

func (v validator) validateGroup(ctx context.Context, model interface{}, group string) []error {
	errs := make([]error, 0)

	if _, ok := v.rules[group]; !ok {
		return errs
	}

	for _, r := range v.rules[group] {
		if err := r.IsValid(ctx, model); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}
