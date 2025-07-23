package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

//go:generate go run github.com/vektra/mockery/v2 --name AuditLogger
type AuditLogger interface {
	ladon.AuditLogger
}

type auditSettings struct {
	LogGrants     bool `cfg:"log_grants" default:"false"`
	LogRejections bool `cfg:"log_rejections" default:"true"`
}

type auditLogger struct {
	logger   log.Logger
	settings auditSettings
}

func NewAuditLogger(config cfg.Config, logger log.Logger) (AuditLogger, error) {
	settings := auditSettings{}
	if err := config.UnmarshalKey("guard.audit", &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal guard audit settings: %w", err)
	}

	return &auditLogger{
		logger:   logger.WithChannel("guard_access"),
		settings: settings,
	}, nil
}

func (a auditLogger) LogRejectedAccessRequest(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	if !a.settings.LogRejections {
		return
	}

	logger := a.logger.
		WithFields(buildLogFields(request, deciders))

	if len(deciders) == 0 {
		logger.Info(ctx, "no policy allowed access for %s on %s", request.Subject, request.Resource)

		return
	}

	rejecter := deciders[len(deciders)-1]

	logger.Info(ctx, "%d policy(s) allow access, but policy %s denied the access for %s on %s", len(deciders)-1, rejecter.GetID(), request.Subject, request.Resource)
}

func (a auditLogger) LogGrantedAccessRequest(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	if !a.settings.LogGrants {
		return
	}

	logger := a.logger.
		WithFields(buildLogFields(request, deciders))

	logger.Info(ctx, "%d policy(s) allow access for %s on %s", len(deciders), request.Subject, request.Resource)
}

func buildLogFields(request *ladon.Request, deciders ladon.Policies) log.Fields {
	//nolint:errcheck // not possible currently due to the interface
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
