package condition

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/tracing"
)

type Condition interface {
	IsValid(ctx context.Context, model db_repo.ModelBased, operation string) error
}

type conditionMap map[string][]Condition

type Conditions interface {
	AddCondition(condition Condition, operations ...string)
	IsValid(ctx context.Context, model db_repo.ModelBased, operation string) error
}

type conditions struct {
	tracer     tracing.Tracer
	conditions conditionMap
}

func NewConditions(config cfg.Config) *conditions {
	tracer := tracing.NewAwsTracer(config)

	return NewConditionsWithInterfaces(tracer)
}

func NewConditionsWithInterfaces(tracer tracing.Tracer) *conditions {
	return &conditions{
		tracer:     tracer,
		conditions: make(conditionMap),
	}
}

func (v *conditions) AddCondition(condition Condition, operations ...string) {
	if len(operations) == 0 {
		operations = []string{
			db_repo.Create,
			db_repo.Update,
			db_repo.Delete,
		}
	}

	for _, o := range operations {
		if _, ok := v.conditions[o]; !ok {
			v.conditions[o] = make([]Condition, 0)
		}

		v.conditions[o] = append(v.conditions[o], condition)
	}
}

func (v conditions) IsValid(ctx context.Context, model db_repo.ModelBased, operation string) error {
	ctx, span := v.tracer.StartSubSpan(ctx, "conditions")
	defer span.Finish()

	conditions, ok := v.conditions[operation]
	if !ok {
		return nil
	}

	for _, r := range conditions {
		if err := r.IsValid(ctx, model, operation); err != nil {
			return err
		}
	}

	return nil
}
