package guard_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/db"
	dbMocks "github.com/justtrackio/gosoline/pkg/db/mocks"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSqlManagerConditionalWrites(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		for _, affected := range []int64{0, 1} {
			t.Run(fmt.Sprintf("%s/%d", operation, affected), func(t *testing.T) {
				client := dbMocks.NewClient(t)
				manager := guard.NewSqlManagerWithInterfaces(log.NewLogger(), client)
				ctx := t.Context()
				// Preserve whitespace and fields not represented in DefaultPolicy.
				raw := `{ "subjects": ["a:3"], "custom": "KEEP" }`
				rows := db.Result{{"id": "shared", "policy": raw}}
				client.EXPECT().GetResult(ctx, "SELECT id, policy FROM guard_policies WHERE id = ?", "shared").Return(&rows, nil).Once()
				before, err := manager.GetSnapshot(ctx, "shared")
				require.NoError(t, err)
				require.Equal(t, "shared", before.Policy.ID)

				if operation == "delete" {
					query := "DELETE FROM guard_policies WHERE id = ? AND BINARY policy = BINARY ?"
					client.EXPECT().Exec(ctx, query, "shared", raw).Return(sqlmock.NewResult(0, affected), nil).Once()
					changed, err := manager.DeleteIfUnchanged(ctx, *before)
					require.NoError(t, err)
					require.Equal(t, affected == 1, changed)
				} else {
					query := "UPDATE guard_policies SET policy = ?, updated_at = ? WHERE id = ? AND BINARY policy = BINARY ?"
					client.EXPECT().Exec(ctx, query, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "shared", raw).
						Return(sqlmock.NewResult(0, affected), nil).Once()
					changed, err := manager.UpdateIfUnchanged(ctx, *before, &ladon.DefaultPolicy{ID: "shared", Subjects: []string{"a:4"}})
					require.NoError(t, err)
					require.Equal(t, affected == 1, changed)
				}
			})
		}
	}
}

func TestSqlManagerSnapshotMissingAndErrors(t *testing.T) {
	client := dbMocks.NewClient(t)
	manager := guard.NewSqlManagerWithInterfaces(log.NewLogger(), client)
	ctx := t.Context()
	rows := db.Result{}
	client.EXPECT().GetResult(ctx, mock.Anything, "missing").Return(&rows, nil).Once()
	before, err := manager.GetSnapshot(ctx, "missing")
	require.NoError(t, err)
	require.Nil(t, before)

	storeErr := errors.New("database unavailable")
	client.EXPECT().GetResult(ctx, mock.Anything, "error").Return(nil, storeErr).Once()
	_, err = manager.GetSnapshot(ctx, "error")
	require.ErrorIs(t, err, storeErr)

	rows = db.Result{{"id": "invalid", "policy": "not json"}}
	client.EXPECT().GetResult(ctx, mock.Anything, "invalid").Return(&rows, nil).Once()
	_, err = manager.GetSnapshot(ctx, "invalid")
	require.ErrorContains(t, err, "could not decode")
}

func TestSqlManagerRejectsInvalidConditionalWrites(t *testing.T) {
	client := dbMocks.NewClient(t)
	manager := guard.NewSqlManagerWithInterfaces(log.NewLogger(), client)
	ctx := t.Context()
	changed, err := manager.DeleteIfUnchanged(ctx, guard.PolicySnapshot{})
	require.Error(t, err)
	require.False(t, changed)

	changed, err = manager.UpdateIfUnchanged(ctx, guard.PolicySnapshot{}, &ladon.DefaultPolicy{ID: "other"})
	require.Error(t, err)
	require.False(t, changed)

	rows := db.Result{{"id": "original", "policy": `{"subjects":["a:3"]}`}}
	client.EXPECT().GetResult(ctx, mock.Anything, "original").Return(&rows, nil).Once()
	before, err := manager.GetSnapshot(ctx, "original")
	require.NoError(t, err)
	before.Policy.ID = "other"
	changed, err = manager.DeleteIfUnchanged(ctx, *before)
	require.Error(t, err)
	require.False(t, changed)
}
