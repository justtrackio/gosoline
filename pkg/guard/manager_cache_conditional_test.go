package guard_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/guard/mocks"
	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/selm0/ladon"
	"github.com/stretchr/testify/require"
)

func TestConditionalWritesClearRequestCacheOnConflict(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			store := mocks.NewManager(t)
			manager := guard.NewCachedManagerWithInterfaces(store)
			ctx := reqctx.New(t.Context())
			before := guard.PolicySnapshot{Policy: &ladon.DefaultPolicy{ID: "shared"}}
			store.EXPECT().Get(ctx, "shared").Return(before.Policy, nil).Twice()
			_, err := manager.Get(ctx, "shared")
			require.NoError(t, err)
			_, err = manager.Get(ctx, "shared")
			require.NoError(t, err)

			var changed bool
			if operation == "update" {
				store.EXPECT().UpdateIfUnchanged(ctx, before, before.Policy).Return(false, nil).Once()
				changed, err = manager.UpdateIfUnchanged(ctx, before, before.Policy)
			} else {
				store.EXPECT().DeleteIfUnchanged(ctx, before).Return(false, nil).Once()
				changed, err = manager.DeleteIfUnchanged(ctx, before)
			}

			require.NoError(t, err)
			require.False(t, changed)
			_, err = manager.Get(ctx, "shared")
			require.NoError(t, err)

			store.EXPECT().GetSnapshot(ctx, "shared").Return(&before, nil).Twice()
			_, err = manager.GetSnapshot(ctx, "shared")
			require.NoError(t, err)
			_, err = manager.GetSnapshot(ctx, "shared")
			require.NoError(t, err)
		})
	}
}
