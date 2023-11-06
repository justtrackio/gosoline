package guard

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/ory/ladon"
)

//go:generate mockery --name AuditLogger
type AuditLogger interface {
	ladon.AuditLogger
}

type auditLogger struct {
	logger log.Logger
}

func NewAuditLogger(logger log.Logger) AuditLogger {
	return &auditLogger{
		logger: logger.WithChannel("guard_access").WithContext(context.Background()),
	}
}

func (a auditLogger) LogRejectedAccessRequest(request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	logger := a.logger.WithFields(buildLogFields(request, deciders))

	if len(deciders) == 0 {
		logger.Info("no policy allowed access for %s on %s", request.Subject, request.Resource)

		return
	}

	rejecter := deciders[len(deciders)-1]

	logger.Info("%d policy(s) allow access, but policy %s denied the access for %s on %s", len(deciders)-1, rejecter.GetID(), request.Subject, request.Resource)
}

func (a auditLogger) LogGrantedAccessRequest(request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	logger := a.logger.WithFields(buildLogFields(request, deciders))

	logger.Info("%d policy(s) allow access for %s on %s", len(deciders), request.Subject, request.Resource)
}

func buildLogFields(request *ladon.Request, deciders ladon.Policies) log.Fields {
	ctx, _ := json.Marshal(request.Context)

	fields := log.Fields{
		"access_resource": request.Resource,
		"access_action":   request.Action,
		"access_subject":  request.Subject,
		"access_context":  string(ctx),
		"access_policy_ids": strings.Join(funk.Map(deciders, func(p ladon.Policy) string {
			return p.GetID()
		}), ", "),
	}

	return fields
}
