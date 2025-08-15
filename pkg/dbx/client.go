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
	Insert(val T) InsertBuilder[T]
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

type client[T any] struct {
	client    db.Client
	table     string
	columns   []string
	arguments []string
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

	return NewClientWithInterfaces[T](client, table), nil
}

// NewClientWithInterfaces creates a new dbx client with a given database client.
// It takes a database client and a table name as arguments.
// It returns a new client.
func NewClientWithInterfaces[T any](dbClient db.Client, table string) *client[T] {
	columns := refl.GetTags(new(T), "db")
	arguments := funk.Map(columns, func(column string) string {
		return ":" + column
	})

	return &client[T]{
		client:    dbClient,
		table:     table,
		columns:   columns,
		arguments: arguments,
	}
}

// Delete creates a new DELETE query builder.
//
//	_, err := client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)
func (c *client[T]) Delete() DeleteBuilder[T] {
	builder.Register(DeleteBuilder[T]{}, deleteData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	db := DeleteBuilder[T](b).from(c.table)
	db = db.placeholderFormat(Question)
	db = builder.Set(db, "Client", c.client).(DeleteBuilder[T])

	return db
}

// Insert creates a new INSERT query builder.
//
//	_, err := client.Insert(YourModel{Id: 1, Name: "test"}).Exec(ctx)
func (c *client[T]) Insert(val T) InsertBuilder[T] {
	builder.Register(InsertBuilder[T]{}, insertData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	ib := InsertBuilder[T](b).into(c.table).columns(c.columns...).value(val)
	ib = builder.Set(ib, "Client", c.client).(InsertBuilder[T])

	return ib
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
	builder.Register(SelectBuilder[T]{}, selectData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	sb := SelectBuilder[T](b).columns(c.columns...).from(c.table)
	sb = sb.placeholderFormat(Question)
	sb = builder.Set(sb, "Client", c.client).(SelectBuilder[T])

	return sb
}

// Update creates a new UPDATE query builder.
//
//	_, err := client.Update(map[string]any{"name": "new_name"}).Where(dbx.Eq{"id": 1}).Exec(ctx)
func (c *client[T]) Update(updateMaps ...any) UpdateBuilder[T] {
	builder.Register(UpdateBuilder[T]{}, updateData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	sb := UpdateBuilder[T](b).table(c.table)
	sb = sb.placeholderFormat(Question)
	sb = builder.Set(sb, "Client", c.client).(UpdateBuilder[T])

	finalMap := make(map[string]any)
	for _, updateMap := range updateMaps {
		switch val := updateMap.(type) {
		case map[string]any:
			finalMap = funk.MergeMaps(finalMap, val)
		case T:
			msi, err := toNonZeroMap(val)
			if err != nil {
				err = fmt.Errorf("unable to convert struct to map: %w", err)
				sb = builder.Set(sb, "Error", err).(UpdateBuilder[T])
				continue
			}
			finalMap = funk.MergeMaps(finalMap, msi)
		}
	}

	return sb.SetMap(finalMap)
}
