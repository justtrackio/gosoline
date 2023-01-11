package limit

import (
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type Invocation interface {
	GetTraceId() string
	GetPrefix() string
	GetName() string
}

type defaultInvocation struct {
	traceId string
	prefix  string
	name    string
}

func (d defaultInvocation) GetTraceId() string {
	return d.traceId
}

func (d defaultInvocation) GetPrefix() string {
	return d.prefix
}

func (d defaultInvocation) GetName() string {
	return d.name
}

type invocationBuilder struct {
	uuid        uuid.Uuid
	limiterName string
}

func (i *invocationBuilder) Build(prefix string) Invocation {
	return defaultInvocation{
		traceId: i.uuid.NewV4(),
		prefix:  prefix,
		name:    i.limiterName,
	}
}

func newInvocationBuilder(limiterName string) (*invocationBuilder, error) {
	return &invocationBuilder{uuid.New(), limiterName}, nil
}
