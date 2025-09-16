package guard_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/guard/mocks"
	"github.com/selm0/ladon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test LadonGuard::GetPolicies
// Testing the other functions does not make any sense here, as they're wrapping just manager methods

func TestLadonGuard_GetPolicies(t *testing.T) {
	manager := mocks.NewManager(t)
	auditLogger := mocks.NewAuditLogger(t)
	g := guard.NewGuardWithInterfaces(manager, auditLogger)

	pol1 := &ladon.DefaultPolicy{
		ID: "100",
	}

	pol2 := &ladon.DefaultPolicy{
		ID: "200",
	}

	ctx := t.Context()

	manager.EXPECT().GetAll(ctx, int64(100), int64(0)).Return(ladon.Policies{pol1}, nil).Once()
	manager.EXPECT().GetAll(ctx, int64(100), int64(100)).Return(ladon.Policies{pol2}, nil).Once()
	manager.EXPECT().GetAll(ctx, int64(100), int64(200)).Return(ladon.Policies{}, nil).Once()

	pols, err := g.GetPolicies(t.Context())
	require.NoError(t, err)

	expected := ladon.Policies{pol1, pol2}
	assert.Equal(t, expected, pols)
}
