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

type Client[T any] interface {
	Delete() DeleteBuilder[T]
	Insert(val T) InsertBuilder[T]
	Replace(val T) InsertBuilder[T]
	Select() SelectBuilder[T]
	Update(updateMaps ...map[string]any) UpdateBuilder[T]
}

type client[T any] struct {
	client    db.Client
	table     string
	columns   []string
	arguments []string
}

func NewClient[T any](ctx context.Context, config cfg.Config, logger log.Logger, clientName string, table string) (*client[T], error) {
	var err error
	var client db.Client

	if client, err = db.ProvideClient(ctx, config, logger, clientName); err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	return NewClientWithInterfaces[T](client, table), nil
}

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

func (c *client[T]) Delete() DeleteBuilder[T] {
	builder.Register(DeleteBuilder[T]{}, deleteData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	db := DeleteBuilder[T](b).from(c.table)
	db = db.placeholderFormat(Question)
	db = builder.Set(db, "Client", c.client).(DeleteBuilder[T])

	return db
}

func (c *client[T]) Insert(val T) InsertBuilder[T] {
	builder.Register(InsertBuilder[T]{}, insertData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	ib := InsertBuilder[T](b).into(c.table).columns(c.columns...).value(val)
	ib = builder.Set(ib, "Client", c.client).(InsertBuilder[T])

	return ib
}

func (c *client[T]) Replace(val T) InsertBuilder[T] {
	rb := c.Insert(val)
	rb = rb.statementKeyword("REPLACE")

	return rb
}

func (c *client[T]) Select() SelectBuilder[T] {
	builder.Register(SelectBuilder[T]{}, selectData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	sb := SelectBuilder[T](b).Columns(c.columns...).From(c.table)
	sb = sb.placeholderFormat(Question)
	sb = builder.Set(sb, "Client", c.client).(SelectBuilder[T])

	return sb
}

func (c *client[T]) Update(updateMaps ...map[string]any) UpdateBuilder[T] {
	builder.Register(UpdateBuilder[T]{}, updateData[T]{})

	b := builder.Builder(builder.EmptyBuilder)
	sb := UpdateBuilder[T](b).Table(c.table)
	sb = sb.placeholderFormat(Question)
	sb = builder.Set(sb, "Client", c.client).(UpdateBuilder[T])

	finalMap := funk.MergeMaps(updateMaps...)
	sb.SetMap(finalMap)

	return sb
}
