package db_repo

// DuplicateKeyBehavior defines how batch insert should handle duplicate key conflicts.
type DuplicateKeyBehavior int

const (
	// DuplicateKeyError causes the insert to fail if a duplicate key is encountered.
	// This is the default behavior - a standard INSERT without any ON DUPLICATE KEY clause.
	DuplicateKeyError DuplicateKeyBehavior = iota
	// DuplicateKeyUpdate uses MySQL's ON DUPLICATE KEY UPDATE to update existing rows.
	// All non-primary-key columns will be updated with the new values.
	DuplicateKeyUpdate
	// DuplicateKeyIgnore uses MySQL's INSERT IGNORE to skip rows with duplicate keys.
	// No error is returned for duplicates, and existing rows remain unchanged.
	DuplicateKeyIgnore
)

// BatchCreateSettings holds configuration for batch create operations.
type BatchCreateSettings struct {
	DuplicateKeyBehavior DuplicateKeyBehavior
}

// BatchCreateOption is a function that configures BatchCreateSettings.
type BatchCreateOption func(*BatchCreateSettings)

// WithOnDuplicateKeyBehavior configures how batch create handles duplicate key conflicts.
// Pass one of the DuplicateKeyBehavior constants:
//   - DuplicateKeyError: Fail on duplicate keys (default, standard INSERT)
//   - DuplicateKeyUpdate: Update existing rows (MySQL ON DUPLICATE KEY UPDATE)
//   - DuplicateKeyIgnore: Skip duplicates silently (MySQL INSERT IGNORE)
func WithOnDuplicateKeyBehavior(behavior DuplicateKeyBehavior) BatchCreateOption {
	return func(s *BatchCreateSettings) {
		s.DuplicateKeyBehavior = behavior
	}
}

// newBatchCreateSettings creates a new BatchCreateSettings with defaults and applies options.
func newBatchCreateSettings(options ...BatchCreateOption) *BatchCreateSettings {
	settings := &BatchCreateSettings{
		DuplicateKeyBehavior: DuplicateKeyError,
	}

	for _, option := range options {
		option(settings)
	}

	return settings
}
