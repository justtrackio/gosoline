package metric

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/justtrackio/gosoline/pkg/appctx"
)

const (
	// SchemaVersion is the version of the metric emission contract this gosoline build implements:
	// metric name formatting, dimension keys and unit representation. Its current value is v2.0 and
	// it is the single source of truth for the value published under MetadataKeySchemaVersion.
	//
	// Increment rules:
	//   - MAJOR: a metric name, dimension key or unit representation is removed or renamed
	//     (MINOR resets to 0), even if the same change also adds something.
	//   - MINOR: purely additive change.
	//   - unchanged: the observable contract is unchanged.
	SchemaVersion = "v2.0"

	// MetadataKeySchemaVersion is the appctx metadata key under which SchemaVersion is published,
	// and therefore exposed by the metadata server's root route, for applications whose metric
	// daemon is enabled.
	MetadataKeySchemaVersion = "metric.schema_version"
)

// ErrSchemaVersionInvalid is returned when a metric schema version does not match v<MAJOR>.<MINOR>.
var ErrSchemaVersionInvalid = errors.New("invalid metric schema version format")

// schemaVersionFormat matches v<MAJOR>.<MINOR>, both components being decimal integers of 1 to 9
// digits without leading zeros unless the component is exactly 0.
var schemaVersionFormat = regexp.MustCompile(`^v(0|[1-9]\d{0,8})\.(0|[1-9]\d{0,8})$`)

// RegisterSchemaVersion publishes SchemaVersion in the appctx metadata carrier of ctx. It is called
// once per application, while an enabled metric daemon module is built.
func RegisterSchemaVersion(ctx context.Context) error {
	return registerSchemaVersion(ctx, SchemaVersion)
}

// IsValidSchemaVersion reports whether version matches v<MAJOR>.<MINOR>.
func IsValidSchemaVersion(version string) bool {
	return schemaVersionFormat.MatchString(version)
}

// registerSchemaVersion is the version parameterised implementation of RegisterSchemaVersion. It
// writes version unchanged and performs exactly one attempt, leaving the carrier untouched if
// version does not match the expected format.
func registerSchemaVersion(ctx context.Context, version string) error {
	if !IsValidSchemaVersion(version) {
		return fmt.Errorf("can not register metric schema version %q: %w", version, ErrSchemaVersionInvalid)
	}

	if err := appctx.MetadataSet(ctx, MetadataKeySchemaVersion, version); err != nil {
		return fmt.Errorf("can not register the metric schema version: %w", err)
	}

	return nil
}
