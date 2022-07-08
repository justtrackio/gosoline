package share

import (
	"fmt"

	"github.com/ory/ladon"
)

func BuildSharePolicy(uuid string, entity Shareable, ownerId uint, actions []string) ladon.Policy {
	return &ladon.DefaultPolicy{
		ID:          uuid,
		Description: fmt.Sprintf("entity %d shared with owner %d", *entity.GetId(), ownerId),
		Subjects: []string{
			fmt.Sprintf("a:%d", ownerId),
		},
		Effect:    ladon.AllowAccess,
		Resources: entity.GetResources(),
		Actions:   actions,
	}
}
