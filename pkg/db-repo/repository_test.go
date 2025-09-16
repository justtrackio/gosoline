package db_repo_test

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"
	"time"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/clock"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

type MyTestModel struct {
	dbRepo.Model
}

const (
	myTestModel = "myTestModel"
	manyToMany  = "manyToMany"
	oneOfMany   = "oneOfMany"
	hasMany     = "hasMany"
)

var MyTestModelMetadata = dbRepo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "myTestModel",
	},
	TableName:  "my_test_models",
	PrimaryKey: "my_test_models.id",
	Mappings: dbRepo.FieldMappings{
		"myTestModel.id":   dbRepo.NewFieldMapping("my_test_models.id"),
		"myTestModel.name": dbRepo.NewFieldMapping("my_test_models.name"),
	},
}

type ManyToMany struct {
	dbRepo.Model
	RelModel []MyTestModel `gorm:"many2many:many_of_manies;" orm:"assoc_update"`
}

var ManyToManyMetadata = dbRepo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "manyToMany",
	},
	TableName:  "many_to_manies",
	PrimaryKey: "many_to_manies.id",
	Mappings: dbRepo.FieldMappings{
		"manyToMany.id": dbRepo.NewFieldMapping("many_to_manies.id"),
	},
}

type OneOfMany struct {
	dbRepo.Model
	MyTestModel   *MyTestModel `gorm:"foreignkey:MyTestModelId"`
	MyTestModelId *uint
}

var OneOfManyMetadata = dbRepo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "oneOfMany",
	},
	TableName:  "one_of_manies",
	PrimaryKey: "one_of_manies.id",
	Mappings: dbRepo.FieldMappings{
		"oneOfMany.id":   dbRepo.NewFieldMapping("one_of_manies.id"),
		"myTestModel.id": dbRepo.NewFieldMapping("one_of_manies.my_test_model_id"),
	},
}

type HasMany struct {
	dbRepo.Model
	Manies []*Ones `gorm:"association_autoupdate:true;association_autocreate:true;association_save_reference:true;" orm:"assoc_update"`
}

var HasManyMetadata = dbRepo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "hasMany",
	},
	TableName:  "has_manies",
	PrimaryKey: "has_manies.id",
	Mappings: dbRepo.FieldMappings{
		"hasMany.id": dbRepo.NewFieldMapping("has_manies.id"),
	},
}

type Ones struct {
	dbRepo.Model
	HasManyId *uint
}

var metadatas = map[string]dbRepo.Metadata{
	"myTestModel": MyTestModelMetadata,
	"manyToMany":  ManyToManyMetadata,
	"oneOfMany":   OneOfManyMetadata,
	"hasMany":     HasManyMetadata,
}

type idMatcher struct{}

func (a idMatcher) Match(id driver.Value) bool {
	return uint(id.(int64)) == *id1 || uint(id.(int64)) == *id42
}

var (
	id1  = mdl.Box(uint(1))
	id42 = mdl.Box(uint(42))
	id24 = mdl.Box(uint(24))
)

func TestRepository_Create(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*MyTestModel](t, now, myTestModel, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` (`id`,`updated_at`,`created_at`) VALUES (?,?,?)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectCommit()

	model := MyTestModel{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` WHERE (`my_test_models`.`id` = ?) ORDER BY `my_test_models`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, &now, model.UpdatedAt, "UpdatedAt should match")
	assert.Equal(t, &now, model.CreatedAt, "CreatedAt should match")
}

func TestRepository_CreateManyToManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*ManyToMany](t, now, manyToMany, false)

	result := goSqlMock.NewResult(0, 1)
	delRes := goSqlMock.NewResult(0, 0)

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `many_to_manies` (`id`,`updated_at`,`created_at`) VALUES (?,?,?)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies` WHERE (`many_to_many_id` IN (?))").WithArgs(id1).WillReturnResult(delRes)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `many_to_manies` WHERE (`many_to_manies`.`id` = ?) ORDER BY `many_to_manies`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`.`id` WHERE (`many_of_manies`.`many_to_many_id` IN (?))").WillReturnRows(rows)

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToMany(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*ManyToMany](t, now, manyToMany, true)

	result := goSqlMock.NewResult(0, 1)
	delRes := goSqlMock.NewResult(0, 0)

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`id`,`updated_at`,`created_at`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec(
		"INSERT INTO `many_of_manies` \\((`my_test_model_id`|`many_to_many_id`),(`many_to_many_id`|`my_test_model_id`)\\) "+
			"SELECT \\?,\\? FROM DUAL WHERE NOT EXISTS \\(SELECT \\* FROM `many_of_manies` "+"WHERE (`my_test_model_id`|`many_to_many_id`) = \\? AND (`my_test_model_id`|`many_to_many_id`) = \\?\\)",
	).WithArgs(idMatcher{}, idMatcher{}, idMatcher{}, idMatcher{}).WillReturnResult(result)

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`my_test_model_id` NOT IN \\(\\?\\)\\) AND \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id42, id1).WillReturnResult(delRes)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE \\(`many_to_manies`\\.`id` = \\?\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`.`many_to_many_id` IN \\(\\?\\)\\)").WillReturnRows(rows)

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: dbRepo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToOneNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*OneOfMany](t, now, oneOfMany, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `one_of_manies` (`id`,`updated_at`,`created_at`,`my_test_model_id`) VALUES (?,?,?,?)").WithArgs(id1, &now, &now, (*uint)(nil)).WillReturnResult(result)
	dbc.ExpectCommit()

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at", "my_test_model_id"}).AddRow(id1, &now, &now, (*uint)(nil))
	dbc.ExpectQuery("SELECT * FROM `one_of_manies` WHERE (`one_of_manies`.`id` = ?) ORDER BY `one_of_manies`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateManyToOne(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*OneOfMany](t, now, oneOfMany, false)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `one_of_manies` (`id`,`updated_at`,`created_at`,`my_test_model_id`) VALUES (?,?,?,?)").WithArgs(id1, &now, &now, id42).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at", "my_test_model_id"}).AddRow(id1, &now, &now, id42)
	dbc.ExpectQuery("SELECT * FROM `one_of_manies` WHERE (`one_of_manies`.`id` = ?) ORDER BY `one_of_manies`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` WHERE (`id` IN (?)) ORDER BY `my_test_models`.`id` ASC").WithArgs(id42).WillReturnRows(rows)

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: dbRepo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateHasManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*HasMany](t, now, hasMany, false)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `has_manies` (`id`,`updated_at`,`created_at`) VALUES (?,?,?)").WithArgs(id1, &now, &now).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `has_manies` WHERE (`has_manies`.`id` = ?) ORDER BY `has_manies`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `ones` WHERE (`has_many_id` IN (?)) ORDER BY `ones`.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_CreateHasMany(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks[*HasMany](t, now, hasMany, false)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `has_manies` (`updated_at`,`created_at`) VALUES (?,?)").WithArgs(&now, &now).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` (`updated_at`,`created_at`,`has_many_id`) VALUES (?,?,?)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), 0).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` (`updated_at`,`created_at`,`has_many_id`) VALUES (?,?,?)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), 0).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 0 AND id NOT IN (0,0)").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `has_manies` WHERE (`has_manies`.`id` = ?) ORDER BY `has_manies`.`id` ASC LIMIT 1").WithArgs(0).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `ones` WHERE (`has_many_id` IN (?)) ORDER BY `ones`.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: dbRepo.Model{},
		Manies: []*Ones{
			{},
			{},
		},
	}

	err := repo.Create(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
}

func TestRepository_Update(t *testing.T) {
	dbc, repo := getMocks[*MyTestModel](t, myTestModel, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `my_test_models` SET `updated_at` = ? WHERE `my_test_models`.`id` = ?").WithArgs(goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` WHERE (`my_test_models`.`id` = ?) ORDER BY `my_test_models`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)

	model := MyTestModel{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToManyNoRelation(t *testing.T) {
	dbc, repo := getMocks[*ManyToMany](t, manyToMany, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at` = ? WHERE `many_to_manies`.`id` = ?").WithArgs(goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE (`many_to_many_id` IN (?))").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `many_to_manies` WHERE (`many_to_manies`.`id` = ?) ORDER BY `many_to_manies`.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`.`id` WHERE (`many_of_manies`.`many_to_many_id` IN (?))").WithArgs(id1).WillReturnRows(rows)

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToMany(t *testing.T) {
	dbc, repo := getMocks[*ManyToMany](t, manyToMany, true)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at` = \\? WHERE `many_to_manies`\\.`id` = \\?").WithArgs(goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec(
		"INSERT INTO `many_of_manies` \\((`my_test_model_id`|`many_to_many_id`),(`many_to_many_id`|`my_test_model_id`)\\) "+
			"SELECT \\?,\\? FROM DUAL WHERE NOT EXISTS \\(SELECT \\* FROM `many_of_manies` "+"WHERE (`my_test_model_id`|`many_to_many_id`) = \\? AND (`my_test_model_id`|`many_to_many_id`) = \\?\\)",
	).WithArgs(idMatcher{}, idMatcher{}, idMatcher{}, idMatcher{}).WillReturnResult(result)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE \\(`my_test_model_id` NOT IN \\(\\?\\)\\) AND \\(`many_to_many_id` IN \\(\\?\\)\\)").WithArgs(id42, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE \\(`many_to_manies`\\.`id` = \\?\\) ORDER BY `many_to_manies`\\.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT \\* FROM `my_test_models` INNER JOIN `many_of_manies` ON `many_of_manies`.`my_test_model_id` = `my_test_models`\\.`id` WHERE \\(`many_of_manies`.`many_to_many_id` IN \\(\\?\\)\\)").WillReturnRows(rows)

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: dbRepo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToOneNoRelation(t *testing.T) {
	dbc, repo := getMocks[*OneOfMany](t, oneOfMany, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at` = ?, `my_test_model_id` = ?  WHERE `one_of_manies`.`id` = ?").WithArgs(goSqlMock.AnyArg(), (*uint)(nil), id1).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `one_of_manies` WHERE (`one_of_manies`.`id` = ?) ORDER BY `one_of_manies`.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		MyTestModel:   nil,
		MyTestModelId: nil,
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateManyToOne(t *testing.T) {
	dbc, repo := getMocks[*OneOfMany](t, oneOfMany, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at` = ?, `my_test_model_id` = ?  WHERE `one_of_manies`.`id` = ?").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectCommit()

	oneOfManyRows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at", "my_test_model_id"}).AddRow(id1, &now, &now, id42)
	myTestModelRows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id42, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `one_of_manies` WHERE (`one_of_manies`.`id` = ?) ORDER BY `one_of_manies`.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(oneOfManyRows)
	dbc.ExpectQuery("SELECT * FROM `my_test_models` WHERE (`id` IN (?)) ORDER BY `my_test_models`.`id` ASC").WithArgs(id42).WillReturnRows(myTestModelRows)

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: dbRepo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_UpdateHasMany(t *testing.T) {
	dbc, repo := getMocks[*HasMany](t, hasMany, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at` = ? WHERE `has_manies`.`id` = ?").WithArgs(goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` (`updated_at`,`created_at`,`has_many_id`) VALUES (?,?,?)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` (`updated_at`,`created_at`,`has_many_id`) VALUES (?,?,?)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` (`updated_at`,`created_at`,`has_many_id`) VALUES (?,?,?)").WithArgs(goSqlMock.AnyArg(), goSqlMock.AnyArg(), *id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1 AND id NOT IN (0,0,0)").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `has_manies` WHERE (`has_manies`.`id` = ?) ORDER BY `has_manies`.`id` ASC LIMIT 1").WithArgs(id1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `ones` WHERE (`has_many_id` IN (?)) ORDER BY `ones`.`id` ASC").WithArgs(id1).WillReturnRows(rows)

	model := HasMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{},
			{},
			{},
		},
	}

	err := repo.Update(t.Context(), &model)
	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
	assert.NoError(t, err)
}

func TestRepository_UpdateHasManyNoRelation(t *testing.T) {
	dbc, repo := getMocks[*HasMany](t, hasMany, false)
	now := time.Unix(1549964818, 0)

	result := goSqlMock.NewResult(0, 1)
	delResult := goSqlMock.NewResult(0, 0)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at` = ? WHERE `has_manies`.`id` = ?").WithArgs(goSqlMock.AnyArg(), id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(delResult)

	rows := goSqlMock.NewRows([]string{"id", "updated_at", "created_at"}).AddRow(id1, &now, &now)
	dbc.ExpectQuery("SELECT * FROM `has_manies` WHERE (`has_manies`.`id` = ?) ORDER BY `has_manies`.`id` ASC LIMIT 1").WithArgs(1).WillReturnRows(rows)
	dbc.ExpectQuery("SELECT * FROM `ones` WHERE (`has_many_id` IN (?)) ORDER BY `ones`.`id` ASC").WillReturnRows(rows)

	model := HasMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Update(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_Delete(t *testing.T) {
	dbc, repo := getMocks[*MyTestModel](t, myTestModel, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `my_test_models`  WHERE `my_test_models`.`id` = ?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := MyTestModel{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToManyNoRelation(t *testing.T) {
	dbc, repo := getMocks[*ManyToMany](t, manyToMany, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE (`many_to_many_id` IN (?))").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_to_manies`  WHERE `many_to_manies`.`id` = ?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToMany(t *testing.T) {
	dbc, repo := getMocks[*ManyToMany](t, manyToMany, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE (`many_to_many_id` IN (?))").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_to_manies`  WHERE `many_to_manies`.`id` = ?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := ManyToMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		RelModel: []MyTestModel{
			{
				Model: dbRepo.Model{
					Id: id42,
				},
			},
		},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToOneNoRelation(t *testing.T) {
	dbc, repo := getMocks[*OneOfMany](t, oneOfMany, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`.`id` = ?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteManyToOne(t *testing.T) {
	dbc, repo := getMocks[*OneOfMany](t, oneOfMany, false)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`.`id` = ?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := OneOfMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		MyTestModel: &MyTestModel{
			Model: dbRepo.Model{
				Id: id42,
			},
		},
		MyTestModelId: id42,
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteHasMany(t *testing.T) {
	dbc, repo := getMocks[*HasMany](t, hasMany, false)

	childResult := goSqlMock.NewResult(0, 0)
	parentResult := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(childResult)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `has_manies`  WHERE `has_manies`.`id` = ?").WithArgs(id1).WillReturnResult(parentResult)
	dbc.ExpectCommit()

	model := HasMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{
				Model: dbRepo.Model{
					Id: id42,
				},
			},
			{
				Model: dbRepo.Model{
					Id: id24,
				},
			},
		},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_DeleteHasManyNoRelation(t *testing.T) {
	dbc, repo := getMocks[*HasMany](t, hasMany, false)

	childResult := goSqlMock.NewResult(0, 0)
	parentResult := goSqlMock.NewResult(0, 1)

	dbc.ExpectExec("DELETE FROM manies WHERE has_many_id = 1").WillReturnResult(childResult)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `has_manies`  WHERE `has_manies`.`id` = ?").WithArgs(id1).WillReturnResult(parentResult)
	dbc.ExpectCommit()

	model := HasMany{
		Model: dbRepo.Model{
			Id: id1,
		},
		Manies: []*Ones{},
	}

	err := repo.Delete(t.Context(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func queryMatcher(t *testing.T, useRegexMatcher bool) goSqlMock.QueryMatcher {
	whiteSpace := regexp.MustCompile(`\s{2,}`)

	return goSqlMock.QueryMatcherFunc(func(expectedSQL string, actualSQL string) error {
		if useRegexMatcher {
			return goSqlMock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
		}

		expectedSQL = whiteSpace.ReplaceAllString(expectedSQL, " ")
		actualSQL = whiteSpace.ReplaceAllString(actualSQL, " ")

		if expectedSQL != actualSQL {
			_, err := fmt.Printf("Failed to match\n\t%s\nwith\n\t%s\n", expectedSQL, actualSQL)
			assert.NoError(t, err)

			return fmt.Errorf(`could not match actual sql: "%s" with expected sql "%s"`, actualSQL, expectedSQL)
		}

		return nil
	})
}

func getMocks[M dbRepo.ModelBased[uint]](t *testing.T, whichMetadata string, useRegexMatcher bool) (goSqlMock.Sqlmock, dbRepo.Repository[uint, M]) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	tracer := tracing.NewLocalTracer()

	db, clientMock, err := goSqlMock.New(goSqlMock.QueryMatcherOption(queryMatcher(t, useRegexMatcher)))
	assert.NoError(t, err)

	orm, err := dbRepo.NewOrmWithInterfaces(db, dbRepo.OrmSettings{
		Driver: "mysql",
	})
	assert.NoError(t, err)

	testClock := clock.NewFakeClock()

	metadata, ok := metadatas[whichMetadata]
	assert.Truef(t, ok, "couldn't find metadata named: %s", whichMetadata)

	repo := dbRepo.NewWithInterfaces[uint, M](logger, tracer, orm, testClock, metadata, dbRepo.CreateModel[M])

	return clientMock, repo
}

func getTimedMocks[M dbRepo.ModelBased[uint]](t *testing.T, time time.Time, whichMetadata string, useRegexMatcher bool) (goSqlMock.Sqlmock, dbRepo.Repository[uint, M]) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	tracer := tracing.NewLocalTracer()

	db, clientMock, err := goSqlMock.New(goSqlMock.QueryMatcherOption(queryMatcher(t, useRegexMatcher)))
	assert.NoError(t, err)

	orm, err := dbRepo.NewOrmWithInterfaces(db, dbRepo.OrmSettings{
		Driver: "mysql",
	})
	assert.NoError(t, err)

	testClock := clock.NewFakeClockAt(time)

	metadata, ok := metadatas[whichMetadata]
	assert.Truef(t, ok, "couldn't find metadata named: %s", whichMetadata)

	repo := dbRepo.NewWithInterfaces[uint, M](logger, tracer, orm, testClock, metadata, dbRepo.CreateModel[M])

	return clientMock, repo
}
