package db_repo_test

import (
	"context"
	"database/sql/driver"
	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/applike/gosoline/pkg/db-repo"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type MyTestModel struct {
	db_repo.Model
}

const (
	myTestModel = "myTestModel"
	manyToMany  = "manyToMany"
	oneOfMany   = "oneOfMany"
	hasMany     = "hasMany"
)

var MyTestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "myTestModel",
	},
	TableName:  "my_test_models",
	PrimaryKey: "my_test_models.id",
	Mappings: db_repo.FieldMappings{
		"myTestModel.id":   db_repo.NewFieldMapping("my_test_models.id"),
		"myTestModel.name": db_repo.NewFieldMapping("my_test_models.name"),
	},
}

type ManyToMany struct {
	db_repo.Model
	RelModel []MyTestModel `gorm:"many2many:many_of_manies;" orm:"assoc_update"`
}

var ManyToManyMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "manyToMany",
	},
	TableName:  "many_to_manies",
	PrimaryKey: "many_to_manies.id",
	Mappings: db_repo.FieldMappings{
		"manyToMany.id": db_repo.NewFieldMapping("many_to_manies.id"),
	},
}

type OneOfMany struct {
	db_repo.Model
	MyTestModel   *MyTestModel `gorm:"foreignkey:MyTestModelId"`
	MyTestModelId *uint
}

var OneOfManyMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "oneOfMany",
	},
	TableName:  "one_of_manies",
	PrimaryKey: "one_of_manies.id",
	Mappings: db_repo.FieldMappings{
		"oneOfMany.id":   db_repo.NewFieldMapping("one_of_manies.id"),
		"myTestModel.id": db_repo.NewFieldMapping("one_of_manies.my_test_model_id"),
	},
}

type HasMany struct {
	db_repo.Model
	Manies []*Ones `gorm:"association_autoupdate:true;association_autocreate:true;association_save_reference:true;" orm:"assoc_update"`
}

var HasManyMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "hasMany",
	},
	TableName:  "has_manies",
	PrimaryKey: "has_manies.id",
	Mappings: db_repo.FieldMappings{
		"hasMany.id": db_repo.NewFieldMapping("has_manies.id"),
	},
}

type Ones struct {
	db_repo.Model
	HasManyId *uint
}

var metadatas = map[string]db_repo.Metadata{
	"myTestModel": MyTestModelMetadata,
	"manyToMany":  ManyToManyMetadata,
	"oneOfMany":   OneOfManyMetadata,
	"hasMany":     HasManyMetadata,
}

type idMatcher struct {
}

func (a idMatcher) Match(id driver.Value) bool {
	return uint(id.(int64)) == *id1 || uint(id.(int64)) == *id42
}

var id1 = mdl.Uint(1)
var id42 = mdl.Uint(42)
var id24 = mdl.Uint(24)

func TestRepository_Create(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`id`,`updated_at`,`created_at`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(id1, &now, &now).WillReturnResult(result)

	model := MyTestModel{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` WHERE `my_test_models`\\.`id` = \\? AND \\(\\(`my_test_models`\\.`id` = 1\\)\\) ORDER BY `my_test_models`\\.`id` ASC LIMIT 1").WillReturnRows(rows)

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, &now, model.UpdatedAt, "UpdatedAt should match")
	assert.Equal(t, &now, model.CreatedAt, "CreatedAt should match")
}

func TestRepository_CreateManyToManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, manyToMany)

	result := goSqlMock.NewResult(0, 1)
	delRes := goSqlMock.NewResult(0, 0)

	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`id`,`updated_at`,`created_at`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM `many_of_manies` WHERE \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id1).WillReturnResult(delRes)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND \\(\\(`many_to_manies`\\.`id` = 1\\)\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`\\.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`\\.`many_to_many_id` IN \\(\\?\\)\\)").WillReturnRows(rows)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToMany(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, manyToMany)

	result := goSqlMock.NewResult(0, 1)
	delRes := goSqlMock.NewResult(0, 0)

	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`id`,`updated_at`,`created_at`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectExec(
		"INSERT INTO `many_of_manies` \\((`my_test_model_id`|`many_to_many_id`),(`many_to_many_id`|`my_test_model_id`)\\) "+
			"SELECT \\?,\\? FROM DUAL WHERE NOT EXISTS \\(SELECT \\* FROM `many_of_manies` "+"WHERE (`my_test_model_id`|`many_to_many_id`) = \\? AND (`my_test_model_id`|`many_to_many_id`) = \\?\\)",
	).WithArgs(idMatcher{}, idMatcher{}, idMatcher{}, idMatcher{}).WillReturnResult(result)

	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`my_test_model_id` NOT IN \\(\\?\\)\\) AND \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id42, id1).WillReturnResult(delRes)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND \\(\\(`many_to_manies`\\.`id` = 1\\)\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`.`many_to_many_id` IN \\(\\?\\)\\)").WillReturnRows(rows)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: db_repo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToOneNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("INSERT INTO `one_of_manies` \\(`id`,`updated_at`,`created_at`,`my_test_model_id`\\) VALUES \\(\\?,\\?,\\?,\\?\\)").WithArgs(id1, &now, &now, (*uint)(nil)).WillReturnResult(result)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at", "my_test_model_id"}).AddRow(id1, &now, &now, (*uint)(nil))
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND \\(\\(`one_of_manies`\\.`id` = 1\\)\\) ORDER BY `one_of_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToOne(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("INSERT INTO `one_of_manies` \\(`id`,`updated_at`,`created_at`,`my_test_model_id`\\) VALUES \\(\\?,\\?,\\?,\\?\\)").WithArgs(id1, &now, &now, id42).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at", "my_test_model_id"}).AddRow(id1, &now, &now, id42)
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND \\(\\(`one_of_manies`\\.`id` = 1\\)\\) ORDER BY `one_of_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` WHERE \\(`id` IN \\(\\?\\)\\) ORDER BY `my_test_models`\\.`id` ASC").WithArgs(id42).WillReturnRows(rows)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: db_repo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateHasManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, hasMany)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)
	dbc.ExpectExec("INSERT INTO `has_manies` \\(`id`,`updated_at`,`created_at`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND \\(\\(`has_manies`\\.`id` = 1\\)\\) ORDER BY `has_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `ones` WHERE \\(`has_many_id` IN \\(\\?\\)\\) ORDER BY `ones`\\.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateHasMany(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, hasMany)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)

	dbc.ExpectExec("INSERT INTO `has_manies` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\)").WithArgs(&now, &now).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), 0).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), 0).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 0 AND id NOT IN \\(0,0\\)").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND \\(\\(`has_manies`\\.`id` = 0\\)\\) ORDER BY `has_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `ones` WHERE \\(`has_many_id` IN \\(\\?\\)\\) ORDER BY `ones`\\.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: db_repo.Model{},
		Manies: []*Ones{
			{},
			{},
		},
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_Update(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("UPDATE `my_test_models` SET `updated_at` = \\?, `created_at` = \\?  WHERE `my_test_models`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` WHERE `my_test_models`\\.`id` = \\? AND \\(\\(`my_test_models`\\.`id` = 1\\)\\) ORDER BY `my_test_models`\\.`id` ASC LIMIT 1").WillReturnRows(rows)

	model := MyTestModel{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToManyNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, manyToMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at` = \\?, `created_at` = \\?  WHERE `many_to_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id1).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND \\(\\(`many_to_manies`\\.`id` = 1\\)\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`.`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id1).WillReturnRows(rows)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToMany(t *testing.T) {
	dbc, repo := getMocks(t, manyToMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at` = \\?, `created_at` = \\?  WHERE `many_to_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectExec(
		"INSERT INTO `many_of_manies` \\((`my_test_model_id`|`many_to_many_id`),(`many_to_many_id`|`my_test_model_id`)\\) "+
			"SELECT \\?,\\? FROM DUAL WHERE NOT EXISTS \\(SELECT \\* FROM `many_of_manies` "+"WHERE (`my_test_model_id`|`many_to_many_id`) = \\? AND (`my_test_model_id`|`many_to_many_id`) = \\?\\)",
	).WithArgs(idMatcher{}, idMatcher{}, idMatcher{}, idMatcher{}).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`my_test_model_id` NOT IN \\(\\?\\)\\) AND \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id42, id1).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND \\(\\(`many_to_manies`\\.`id` = 1\\)\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`.`many_to_many_id` IN \\(\\?\\)\\)").WillReturnRows(rows)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: db_repo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToOneNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, oneOfMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at` = \\?, `created_at` = \\?, `my_test_model_id` = \\?  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), (*uint)(nil), id1).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND \\(\\(`one_of_manies`\\.`id` = 1\\)\\) ORDER BY `one_of_manies`\\.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
		MyTestModel:   nil,
		MyTestModelId: nil,
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToOne(t *testing.T) {
	dbc, repo := getMocks(t, oneOfMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at` = \\?, `created_at` = \\?, `my_test_model_id` = \\?  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND \\(\\(`one_of_manies`\\.`id` = 1\\)\\) ORDER BY `one_of_manies`\\.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` WHERE \\(`id` IN \\(\\?\\)\\) ORDER BY `my_test_models`\\.`id` ASC").WithArgs(id42).WillReturnRows(rows)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: db_repo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateHasMany(t *testing.T) {
	dbc, repo := getMocks(t, hasMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)
	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at` = \\?, `created_at` = \\?  WHERE `has_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1 AND id NOT IN \\(0,0,0\\)").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND \\(\\(`has_manies`\\.`id` = 1\\)\\) ORDER BY `has_manies`\\.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `ones` WHERE \\(`has_many_id` IN \\(\\?\\)\\) ORDER BY `ones`\\.`id` ASC").WithArgs(id1).WillReturnRows(rows)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{},
			{},
			{},
		},
	}

	err := repo.Update(context.Background(), &model)
	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
	assert.NoError(t, err)
}

func TestRepository_UpdateHasManyNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, hasMany)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)

	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at` = \\?, `created_at` = \\?  WHERE `has_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND \\(\\(`has_manies`\\.`id` = 1\\)\\) ORDER BY `has_manies`\\.`id` ASC LIMIT 1").WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `ones` WHERE \\(`has_many_id` IN \\(\\?\\)\\) ORDER BY `ones`\\.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_Delete(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("DELETE FROM `my_test_models`  WHERE `my_test_models`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)

	model := MyTestModel{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToManyNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, manyToMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM `many_to_manies`  WHERE `many_to_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToMany(t *testing.T) {
	dbc, repo := getMocks(t, manyToMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectExec("DELETE FROM `many_to_manies`  WHERE `many_to_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)

	model := ManyToMany{
		Model: db_repo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: db_repo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToOneNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, oneOfMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToOne(t *testing.T) {
	dbc, repo := getMocks(t, oneOfMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)

	model := OneOfMany{
		Model: db_repo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: db_repo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteHasMany(t *testing.T) {
	dbc, repo := getMocks(t, hasMany)

	childResult := goSqlMock.NewResult(0, 0)
	parentResult := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(childResult)
	dbc.ExpectExec("DELETE FROM `has_manies`  WHERE `has_manies`\\.`id` = ?").WithArgs(id1).WillReturnResult(parentResult)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{
				Model: db_repo.Model{
					Id: id42,
				},
			},
			{
				Model: db_repo.Model{
					Id: id24,
				},
			},
		},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteHasManyNoRelation(t *testing.T) {
	dbc, repo := getMocks(t, hasMany)

	childResult := goSqlMock.NewResult(0, 0)
	parentResult := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(childResult)
	dbc.ExpectExec("DELETE FROM `has_manies`  WHERE `has_manies`\\.`id` = ?").WithArgs(id1).WillReturnResult(parentResult)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Delete(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func getMocks(t *testing.T, whichMetadata string) (goSqlMock.Sqlmock, db_repo.Repository) {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()

	db, clientMock, _ := goSqlMock.New()
	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	clock := clockwork.NewFakeClock()

	metadata, ok := metadatas[whichMetadata]
	if !ok {
		t.Errorf("couldn't find metadata named: %s", whichMetadata)
	}

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, clock, metadata)

	return clientMock, repo
}

func getTimedMocks(t *testing.T, time time.Time, whichMetadata string) (goSqlMock.Sqlmock, db_repo.Repository) {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()

	db, clientMock, _ := goSqlMock.New()

	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	clock := clockwork.NewFakeClockAt(time)

	metadata, ok := metadatas[whichMetadata]
	if !ok {
		t.Errorf("couldn't find metadata named: %s", whichMetadata)
	}

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, clock, metadata)

	return clientMock, repo
}
