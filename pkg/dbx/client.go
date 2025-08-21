package dbx

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/lann/builder"
)

// Client is the main entry point to the package. It is used to create query builders.
type Client[T any] interface {
	// Delete creates a new DELETE query builder.
	//
	//	_, err := client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)
	Delete() DeleteBuilder[T]
	// Insert creates a new INSERT query builder.
	//
	//	_, err := client.Insert(YourModel{Id: 1, Name: "test"}).Exec(ctx)
	Insert(val ...T) InsertBuilder[T]
	// Replace creates a new REPLACE query builder.
	//
	//	_, err := client.Replace(YourModel{Id: 1, Name: "test"}).Exec(ctx)
	Replace(val T) InsertBuilder[T]
	// Select creates a new SELECT query builder.
	//
	//	results, err := client.Select().Where(dbx.Eq{"id": 1}).Exec(ctx)
	Select() SelectBuilder[T]
	// Update creates a new UPDATE query builder.
	//
	//	_, err := client.Update(map[string]any{"name": "new_name"}).Where(dbx.Eq{"id": 1}).Exec(ctx)
	Update(updateMaps ...any) UpdateBuilder[T]
}

var _ Client[any] = (*client[any])(nil)

type client[T any] struct {
	client            db.Client
	table             string
	placeholderFormat PlaceholderFormat
	columns           []string
}

// NewClient creates a new dbx client.
// It takes a context, a config, a logger, a client name and a table name as arguments.
// The client name is the name of the database client to use, as configured in your application's configuration file.
// The table name is the name of the database table.
// It returns a new client or an error if the client could not be created.
func NewClient[T any](ctx context.Context, config cfg.Config, logger log.Logger, clientName string, table string) (*client[T], error) {
	var err error
	var client db.Client

	if client, err = db.ProvideClient(ctx, config, logger, clientName); err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	return NewClientWithInterfaces[T](client, table, Question)
}

// NewClientWithInterfaces creates a new dbx client with a given database client.
// It takes a database client and a table name as arguments.
// It returns a new client.
func NewClientWithInterfaces[T any](dbClient db.Client, table string, placeholderFormat PlaceholderFormat) (*client[T], error) {
	columns := refl.GetTags(new(T), "db")

	if len(columns) == 0 {
		return nil, fmt.Errorf("no db tags found in struct %T", new(T))
	}

	builder.Register(DeleteBuilder[T]{}, deleteData[T]{})
	builder.Register(InsertBuilder[T]{}, insertData[T]{})
	builder.Register(SelectBuilder[T]{}, selectData[T]{})
	builder.Register(UpdateBuilder[T]{}, updateData[T]{})

	return &client[T]{
		client:            dbClient,
		table:             table,
		placeholderFormat: placeholderFormat,
		columns:           columns,
	}, nil
}

// Delete creates a new DELETE query builder.
//
//	_, err := client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)
func (c *client[T]) Delete() DeleteBuilder[T] {
	return newDeleteBuilder[T](c.client, c.table, c.placeholderFormat)
}

// Insert creates a new INSERT query builder.
//
//	_, err := client.Insert(YourModel{Id: 1, Name: "test"}).Exec(ctx)
func (c *client[T]) Insert(values ...T) InsertBuilder[T] {
	return newInsertBuilder[T](c.client, c.table).columns(c.columns...).values(values...)
}

// Replace creates a new REPLACE query builder.
//
//	_, err := client.Replace(YourModel{Id: 1, Name: "test"}).Exec(ctx)
func (c *client[T]) Replace(val T) InsertBuilder[T] {
	rb := c.Insert(val)
	rb = rb.statementKeyword("REPLACE")

	return rb
}

// Select creates a new SELECT query builder.
//
//	results, err := client.Select().Where(dbx.Eq{"id": 1}).Exec(ctx)
func (c *client[T]) Select() SelectBuilder[T] {
	return newSelectBuilder[T](c.client, c.table, c.placeholderFormat).columns(c.columns...)
}

// Update creates a new UPDATE query builder.
//
//	_, err := client.Update(map[string]any{"name": "new_name"}).Where(dbx.Eq{"id": 1}).Exec(ctx)
func (c *client[T]) Update(updateValues ...any) UpdateBuilder[T] {
	ub := newUpdateBuilder[T](c.client, c.table, c.placeholderFormat)

	finalMap := make(map[string]any)
	for _, updateMap := range updateValues {
		switch val := updateMap.(type) {
		case map[string]any:
			finalMap = funk.MergeMaps(finalMap, val)
		case T:
			msi, err := toNonZeroMap(val)
			if err != nil {
				err = fmt.Errorf("unable to convert struct to map: %w", err)
				ub = builder.Set(ub, "Error", err).(UpdateBuilder[T])

				continue
			}
			finalMap = funk.MergeMaps(finalMap, msi)
		default:
			err := fmt.Errorf("unsupported type %T for update values", val)
			ub = builder.Set(ub, "Error", err).(UpdateBuilder[T])
		}
	}

	return ub.SetMap(finalMap)
}
