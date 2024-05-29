package db_repo

import (
	"context"
	"database/sql"
	"reflect"
	"time"
	"unsafe"

	atomicGorm "github.com/beeemT/go-atomic/generic/gorm"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

var (
	_ Remote                        = (*gorm.DB)(nil)
	_ atomicGorm.GormlikeDB[Remote] = &Gorm{}
)

type (
	Gorm struct {
		*gorm.DB
	}

	Remote interface {
		AddError(err error) error
		AddForeignKey(field string, dest string, onDelete string, onUpdate string) *gorm.DB
		AddIndex(indexName string, columns ...string) *gorm.DB
		AddUniqueIndex(indexName string, columns ...string) *gorm.DB
		Assign(attrs ...interface{}) *gorm.DB
		Association(column string) *gorm.Association
		Attrs(attrs ...interface{}) *gorm.DB
		AutoMigrate(values ...interface{}) *gorm.DB
		BeginTx(ctx context.Context, opts *sql.TxOptions) *gorm.DB
		BlockGlobalUpdate(enable bool) *gorm.DB
		Callback() *gorm.Callback
		Close() error
		CommonDB() gorm.SQLCommon
		Count(value interface{}) *gorm.DB
		Create(value interface{}) *gorm.DB
		CreateTable(models ...interface{}) *gorm.DB
		DB() *sql.DB
		Debug() *gorm.DB
		Delete(value interface{}, where ...interface{}) *gorm.DB
		Dialect() gorm.Dialect
		DropColumn(column string) *gorm.DB
		DropTable(values ...interface{}) *gorm.DB
		DropTableIfExists(values ...interface{}) *gorm.DB
		Exec(sql string, values ...interface{}) *gorm.DB
		Find(out interface{}, where ...interface{}) *gorm.DB
		First(out interface{}, where ...interface{}) *gorm.DB
		FirstOrCreate(out interface{}, where ...interface{}) *gorm.DB
		FirstOrInit(out interface{}, where ...interface{}) *gorm.DB
		Get(name string) (value interface{}, ok bool)
		GetErrors() []error
		Group(query string) *gorm.DB
		HasBlockGlobalUpdate() bool
		HasTable(value interface{}) bool
		Having(query interface{}, values ...interface{}) *gorm.DB
		InstantSet(name string, value interface{}) *gorm.DB
		Joins(query string, args ...interface{}) *gorm.DB
		Last(out interface{}, where ...interface{}) *gorm.DB
		Limit(limit interface{}) *gorm.DB
		LogMode(enable bool) *gorm.DB
		Model(value interface{}) *gorm.DB
		ModifyColumn(column string, typ string) *gorm.DB
		New() *gorm.DB
		NewRecord(value interface{}) bool
		NewScope(value interface{}) *gorm.Scope
		Not(query interface{}, args ...interface{}) *gorm.DB
		Offset(offset interface{}) *gorm.DB
		Omit(columns ...string) *gorm.DB
		Or(query interface{}, args ...interface{}) *gorm.DB
		Order(value interface{}, reorder ...bool) *gorm.DB
		Pluck(column string, value interface{}) *gorm.DB
		Preload(column string, conditions ...interface{}) *gorm.DB
		Preloads(out interface{}) *gorm.DB
		QueryExpr() *gorm.SqlExpr
		Raw(sql string, values ...interface{}) *gorm.DB
		RecordNotFound() bool
		Related(value interface{}, foreignKeys ...string) *gorm.DB
		RemoveForeignKey(field string, dest string) *gorm.DB
		RemoveIndex(indexName string) *gorm.DB
		RollbackUnlessCommitted() *gorm.DB
		Row() *sql.Row
		Rows() (*sql.Rows, error)
		Save(value interface{}) *gorm.DB
		Scan(dest interface{}) *gorm.DB
		ScanRows(rows *sql.Rows, result interface{}) error
		Scopes(funcs ...func(*gorm.DB) *gorm.DB) *gorm.DB
		Select(query interface{}, args ...interface{}) *gorm.DB
		Set(name string, value interface{}) *gorm.DB
		SetJoinTableHandler(source interface{}, column string, handler gorm.JoinTableHandlerInterface)
		SetNowFuncOverride(nowFuncOverride func() time.Time) *gorm.DB
		SingularTable(enable bool)
		SubQuery() *gorm.SqlExpr
		Table(name string) *gorm.DB
		Take(out interface{}, where ...interface{}) *gorm.DB
		Transaction(fc func(tx *gorm.DB) error) (err error)
		Unscoped() *gorm.DB
		Update(attrs ...interface{}) *gorm.DB
		UpdateColumn(attrs ...interface{}) *gorm.DB
		UpdateColumns(values interface{}) *gorm.DB
		Updates(values interface{}, ignoreProtectedAttrs ...bool) *gorm.DB
		Where(query interface{}, args ...interface{}) *gorm.DB
	}

	contextWrapperDB struct{}
)

func (g *Gorm) Remote() Remote {
	return g.DB
}

func (g *Gorm) Begin(opts ...*sql.TxOptions) atomicGorm.GormlikeDB[Remote] {
	var opt *sql.TxOptions

	if len(opts) > 0 {
		opt = opts[0]
	}

	return &Gorm{
		DB: g.BeginTx(context.Background(), opt),
	}
}

func (g *Gorm) Commit() atomicGorm.GormlikeDB[Remote] {
	return &Gorm{
		DB: g.DB.Commit(),
	}
}

func (g *Gorm) Error() error {
	return g.DB.Error
}

func (g *Gorm) Rollback() atomicGorm.GormlikeDB[Remote] {
	return &Gorm{
		DB: g.DB.Rollback(),
	}
}

func RemoteToClientBase(remote Remote, logger log.Logger) db.ClientBase {
	var client db.ClientBase
	commonDB := remote.CommonDB()
	dialect := remote.Dialect()
	switch dbCast := commonDB.(type) {
	case *sql.DB:
		client = db.NewClientBaseWithInterfaces(logger, sqlx.NewDb(dbCast, dialect.GetName()))
	case *sql.Tx:
		tx := sqlx.Tx{
			Tx:     dbCast,
			Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper),
		}

		field := reflect.ValueOf(&tx).Elem().FieldByName("driverName")
		if !field.CanSet() { // this should always be true since driverName is unexported
			field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		}
		field.SetString(dialect.GetName())

		client = db.NewClientBaseWithInterfaces(logger, &tx)
	}

	return client
}
