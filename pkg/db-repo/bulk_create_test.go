package db_repo

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bulkCreateTestModel struct {
	Model
	Name string
}

func TestExtractMapValue_WithExplicitId(t *testing.T) {
	// When fixture has explicit ID set, it should be included in the INSERT
	id := uint(42)
	model := bulkCreateTestModel{
		Model: Model{Id: &id},
		Name:  "Test",
	}

	attrs, err := extractMapValue(model, nil)
	require.NoError(t, err)

	// Verify that 'id' IS included when explicit ID is set
	assert.Contains(t, attrs, "id", "id field should be included when explicit ID is set")
	assert.Contains(t, attrs, "name")
}

func TestExtractMapValue_WithoutId(t *testing.T) {
	// When fixture has no ID set (nil), it should NOT be included - let DB auto-generate
	model := bulkCreateTestModel{
		Model: Model{Id: nil},
		Name:  "Test",
	}

	attrs, err := extractMapValue(model, nil)
	require.NoError(t, err)

	// Verify that 'id' is NOT included when ID is nil (DB will auto-generate)
	assert.NotContains(t, attrs, "id", "id field should NOT be included when ID is nil")
	assert.Contains(t, attrs, "name")
}

func TestExtractMapValue_FieldIsAutoIncrementAndBlank(t *testing.T) {
	// Verify the IsBlank detection works correctly for pointer fields
	idSet := uint(1)

	tests := []struct {
		name        string
		model       bulkCreateTestModel
		expectBlank bool
	}{
		{
			name:        "nil pointer is blank",
			model:       bulkCreateTestModel{Model: Model{Id: nil}, Name: "Test"},
			expectBlank: true,
		},
		{
			name:        "non-nil pointer is not blank",
			model:       bulkCreateTestModel{Model: Model{Id: &idSet}, Name: "Test"},
			expectBlank: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := &gorm.Scope{Value: tt.model}
			for _, field := range scope.Fields() {
				if field.Struct.Name == "Id" {
					assert.Equal(t, tt.expectBlank, field.IsBlank, "IsBlank mismatch for Id field")
					assert.True(t, fieldIsAutoIncrement(field), "Id should be AUTO_INCREMENT")

					isAutoIncrementAndBlank := fieldIsAutoIncrement(field) && field.IsBlank
					assert.Equal(t, tt.expectBlank, isAutoIncrementAndBlank)
				}
			}
		})
	}
}

func TestSplitObjects(t *testing.T) {
	tests := []struct {
		name     string
		objects  []any
		size     int
		expected [][]any
	}{
		{
			name:     "empty slice",
			objects:  []any{},
			size:     2,
			expected: nil,
		},
		{
			name:     "single chunk",
			objects:  []any{1, 2},
			size:     3,
			expected: [][]any{{1, 2}},
		},
		{
			name:     "exact chunks",
			objects:  []any{1, 2, 3, 4},
			size:     2,
			expected: [][]any{{1, 2}, {3, 4}},
		},
		{
			name:     "remainder chunk",
			objects:  []any{1, 2, 3, 4, 5},
			size:     2,
			expected: [][]any{{1, 2}, {3, 4}, {5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitObjects(tt.objects, tt.size)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortedKeys(t *testing.T) {
	input := map[string]any{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
	}

	result := sortedKeys(input)

	assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
}

func TestContainString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, containString(slice, "banana"))
	assert.False(t, containString(slice, "grape"))
	assert.False(t, containString(nil, "apple"))
	assert.False(t, containString([]string{}, "apple"))
}
