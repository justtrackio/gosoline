package share

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/selm0/ladon"
)

func BuildSharePolicy[K mdl.PossibleIdentifier](uuid string, entity Shareable[K], ownerId uint, actions []string) ladon.Policy {
	return &ladon.DefaultPolicy{
		ID:          uuid,
		Description: fmt.Sprintf("entity %v shared with owner %d", *entity.GetId(), ownerId),
		Subjects: []string{
			fmt.Sprintf("a:%d", ownerId),
		},
		Effect:    ladon.AllowAccess,
		Resources: entity.GetResources(),
		Actions:   actions,
	}
}
