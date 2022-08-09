package db_repo_test

import (
	"context"
	"testing"
	"time"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
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
	RelModel []MyTestModel `gorm:"many2many:many_of_manies" orm:"assoc_update"`
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
	MyTestModel   *MyTestModel `gorm:"foreignKey:MyTestModelId"`
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
	Manies []*Ones `orm:"assoc_update"`
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

var (
	id1  = mdl.Box(uint(1))
	id6  = mdl.Box(uint(6))
	id42 = mdl.Box(uint(42))
	id24 = mdl.Box(uint(24))
)

func TestRepository_BatchCreate(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(1, 2)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\),\\(\\?,\\?\\)").WithArgs(&now, &now, &now, &now).WillReturnResult(result)
	dbc.ExpectCommit()

	models := []*MyTestModel{
		{
			Model: db_repo.Model{},
		},
		{
			Model: db_repo.Model{},
		},
	}

	err := repo.BatchCreate(context.Background(), &models)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, uint(1), mdl.EmptyIfNil(models[0].Id), "id should match for model 0")
	assert.Equal(t, &now, models[0].UpdatedAt, "UpdatedAt should match for model 0")
	assert.Equal(t, &now, models[0].CreatedAt, "CreatedAt should match for model 0")
	assert.Equal(t, uint(2), mdl.EmptyIfNil(models[1].Id), "id should match for model 1")
	assert.Equal(t, &now, models[1].UpdatedAt, "UpdatedAt should match for model 1")
	assert.Equal(t, &now, models[1].CreatedAt, "CreatedAt should match for model 1")
}

func TestRepository_BatchUpdate(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 2)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\),\\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1, &now, &now, id42).WillReturnResult(result)
	dbc.ExpectCommit()

	models := []*MyTestModel{
		{
			Model: db_repo.Model{
				Id: id1,
			},
		},
		{
			Model: db_repo.Model{
				Id: id42,
			},
		},
	}

	err := repo.BatchUpdate(context.Background(), &models)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, *id1, mdl.EmptyIfNil(models[0].Id), "id should match for model 0")
	assert.Equal(t, &now, models[0].UpdatedAt, "UpdatedAt should match for model 0")
	assert.Equal(t, &now, models[0].CreatedAt, "CreatedAt should match for model 0")
	assert.Equal(t, *id42, mdl.EmptyIfNil(models[1].Id), "id should match for model 1")
	assert.Equal(t, &now, models[1].UpdatedAt, "UpdatedAt should match for model 1")
	assert.Equal(t, &now, models[1].CreatedAt, "CreatedAt should match for model 1")
}

func TestRepository_BatchDelete(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 2)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `my_test_models` WHERE `my_test_models`\\.`id` IN \\(\\?,\\?\\)").WithArgs(id1, id42).WillReturnResult(result)
	dbc.ExpectCommit()

	models := []*MyTestModel{
		{
			Model: db_repo.Model{
				Id: id1,
			},
		},
		{
			Model: db_repo.Model{
				Id: id42,
			},
		},
	}

	err := repo.BatchDelete(context.Background(), &models)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
}

func TestRepository_Create(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := MyTestModel{
		Model: db_repo.Model{
			Id: id1,
		},
	}

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

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

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

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id42).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `many_of_manies` \\(`many_to_many_id`,`my_test_model_id`\\) VALUES \\(\\?,\\?\\) ON DUPLICATE KEY UPDATE `many_to_many_id`=`many_to_many_id`").WithArgs(id1, id42).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `one_of_manies` \\(`updated_at`,`created_at`,`my_test_model_id`,`id`\\) VALUES \\(\\?,\\?,\\?,\\?\\)").WithArgs(&now, &now, (*uint)(nil), id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := OneOfMany{
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

func TestRepository_CreateManyToOne(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `id`=`id`").WithArgs(&now, &now, id42).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `one_of_manies` \\(`updated_at`,`created_at`,`my_test_model_id`,`id`\\) VALUES \\(\\?,\\?,\\?,\\?\\)").WithArgs(&now, &now, id42, id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `has_manies` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

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

	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `has_manies` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`,`id`\\) VALUES \\(\\?,\\?,\\?,\\?\\),\\(\\?,\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `has_many_id`=VALUES\\(`has_many_id`\\)").WithArgs(&now, &now, id1, id42, &now, &now, id1, id24).WillReturnResult(result)
	dbc.ExpectCommit()

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{
				Model: db_repo.Model{
					Id: id42,
				},
				HasManyId: id1,
			},
			{
				Model: db_repo.Model{
					Id: id24,
				},
				HasManyId: id1,
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

func TestRepository_Create_NoPrimary(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(1, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\)").WithArgs(&now, &now).WillReturnResult(result)
	dbc.ExpectCommit()

	model := MyTestModel{Model: db_repo.Model{}}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, &now, model.UpdatedAt, "UpdatedAt should match")
	assert.Equal(t, &now, model.CreatedAt, "CreatedAt should match")
	assert.Equal(t, uint(1), *model.Id)
}

func TestRepository_CreateManyToMany_NoPrimary(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, manyToMany)

	dbc.ExpectBegin()
	resultManyToManies := goSqlMock.NewResult(int64(*id1), 1)
	dbc.ExpectExec("INSERT INTO `many_to_manies` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\)").WithArgs(&now, &now).WillReturnResult(resultManyToManies)
	resultMyTestModels := goSqlMock.NewResult(int64(*id42), 1)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\)").WithArgs(&now, &now).WillReturnResult(resultMyTestModels)
	resultManyOfManies := goSqlMock.NewResult(0, 1)
	dbc.ExpectExec("INSERT INTO `many_of_manies` \\(`many_to_many_id`,`my_test_model_id`\\) VALUES \\(\\?,\\?\\) ON DUPLICATE KEY UPDATE `many_to_many_id`=`many_to_many_id`").WithArgs(id1, id42).WillReturnResult(resultManyOfManies)
	dbc.ExpectCommit()

	model := ManyToMany{
		Model: db_repo.Model{},
		RelModel: []MyTestModel{
			{
				Model: db_repo.Model{},
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
	assert.Equal(t, *id1, *model.Id)
	assert.Equal(t, *id42, *model.RelModel[0].Id)
}

func TestRepository_CreateManyToOne_NoPrimary(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	dbc.ExpectBegin()
	resultMyTestModels := goSqlMock.NewResult(int64(*id42), 1)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\) ON DUPLICATE KEY UPDATE `id`=`id`").WithArgs(&now, &now).WillReturnResult(resultMyTestModels)
	resultOneOfManies := goSqlMock.NewResult(int64(*id1), 1)
	dbc.ExpectExec("INSERT INTO `one_of_manies` \\(`updated_at`,`created_at`,`my_test_model_id`\\) VALUES \\(\\?,\\?,\\?\\)").WithArgs(&now, &now, id42).WillReturnResult(resultOneOfManies)
	dbc.ExpectCommit()

	model := OneOfMany{
		Model: db_repo.Model{},
		MyTestModel: &MyTestModel{
			Model: db_repo.Model{},
		},
	}

	err := repo.Create(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &now, model.UpdatedAt)
	assert.Equal(t, &now, model.CreatedAt)
	assert.Equal(t, *id1, *model.Id)
	assert.Equal(t, *id42, *model.MyTestModel.Id)
	assert.Equal(t, *id42, *model.MyTestModelId)
}

func TestRepository_CreateHasMany_NoPrimary(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, hasMany)

	dbc.ExpectBegin()
	resultHasManies := goSqlMock.NewResult(int64(*id1), 1)
	dbc.ExpectExec("INSERT INTO `has_manies` \\(`updated_at`,`created_at`\\) VALUES \\(\\?,\\?\\)").WithArgs(&now, &now).WillReturnResult(resultHasManies)
	resultOnes := goSqlMock.NewResult(int64(*id42), 1)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`\\) VALUES \\(\\?,\\?,\\?\\),\\(\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `has_many_id`=VALUES\\(`has_many_id`\\)").WithArgs(&now, &now, id1, &now, &now, id1).WillReturnResult(resultOnes)
	dbc.ExpectCommit()

	model := HasMany{
		Model: db_repo.Model{},
		Manies: []*Ones{
			{
				Model: db_repo.Model{},
			},
			{
				Model: db_repo.Model{},
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
	assert.Equal(t, *id1, *model.Id)
	assert.Equal(t, *id42, *model.Manies[0].Id)
	assert.Equal(t, *id42+1, *model.Manies[1].Id)
}

func TestRepository_Update(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `my_test_models` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	model := MyTestModel{
		Model: db_repo.Model{
			Id: id1,
			Timestamps: db_repo.Timestamps{
				CreatedAt: &now,
			},
		},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
}

func TestRepository_Update_MissingCreatedAt(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, myTestModel)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `my_test_models` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	assert.Equal(t, &now, model.CreatedAt)
	assert.Equal(t, &now, model.UpdatedAt)
}

func TestRepository_UpdateManyToManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, manyToMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	deleteResult := goSqlMock.NewResult(0, 0)
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies` WHERE `many_of_manies`\\.`many_to_many_id` = \\?").WithArgs(id1).WillReturnResult(deleteResult)
	dbc.ExpectCommit()

	queryResult := goSqlMock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(*id1, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND `many_to_manies`\\.`id` = \\? ORDER BY `many_to_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(queryResult)

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
	assert.Equal(t, *id1, *model.Id)
	assert.Equal(t, now, *model.CreatedAt)
	assert.Equal(t, now, *model.UpdatedAt)
}

func TestRepository_UpdateManyToMany(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, manyToMany)

	result := goSqlMock.NewResult(0, 2)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\),\\(\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `id`=`id`").WithArgs(&now, &now, id42, &now, &now, id24).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `many_of_manies` \\(`many_to_many_id`,`my_test_model_id`\\) VALUES \\(\\?,\\?\\),\\(\\?,\\?\\) ON DUPLICATE KEY UPDATE `many_to_many_id`=`many_to_many_id`").WithArgs(id1, id42, id1, id24).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `many_to_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\),\\(\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `id`=`id`").WithArgs(&now, &now, id42, &now, &now, id24).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `many_of_manies` \\(`many_to_many_id`,`my_test_model_id`\\) VALUES \\(\\?,\\?\\),\\(\\?,\\?\\) ON DUPLICATE KEY UPDATE `many_to_many_id`=`many_to_many_id`").WithArgs(id1, id42, id1, id24).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies` WHERE `many_of_manies`\\.`many_to_many_id` = \\? AND `many_of_manies`\\.`my_test_model_id` NOT IN \\(\\?,\\?\\)").WithArgs(id1, id42, id24).WillReturnResult(result)
	dbc.ExpectCommit()

	rows := goSqlMock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(*id1, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\? AND `many_to_manies`\\.`id` = \\? ORDER BY `many_to_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(rows)

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
			{
				Model: db_repo.Model{
					Id: id24,
				},
			},
		},
	}

	err := repo.Update(context.Background(), &model)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.NoError(t, err)
	assert.Len(t, model.RelModel, 2)
}

func TestRepository_UpdateManyToOneNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at`=\\?,`created_at`=\\?,`my_test_model_id`=\\? WHERE `id` = \\?").WithArgs(&now, &now, (*uint)(nil), id1).WillReturnResult(result)
	dbc.ExpectCommit()

	queryResult := goSqlMock.NewRows([]string{"id", "my_test_model_id", "created_at", "updated_at"}).AddRow(*id1, nil, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND `one_of_manies`\\.`id` = \\? ORDER BY `one_of_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(queryResult)

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
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, oneOfMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("INSERT INTO `my_test_models` \\(`updated_at`,`created_at`,`id`\\) VALUES \\(\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `id`=`id`").WithArgs(&now, &now, id42).WillReturnResult(result)
	dbc.ExpectExec("UPDATE `one_of_manies` SET `updated_at`=\\?,`created_at`=\\?,`my_test_model_id`=\\?  WHERE `id` = \\?").WithArgs(&now, &now, id42, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	queryResult := goSqlMock.NewRows([]string{"id", "my_test_model_id", "created_at", "updated_at"}).AddRow(*id1, nil, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `one_of_manies` WHERE `one_of_manies`\\.`id` = \\? AND `one_of_manies`\\.`id` = \\? ORDER BY `one_of_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(queryResult)

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
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, hasMany)

	result := goSqlMock.NewResult(0, 1)
	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectExec("INSERT INTO `ones` \\(`updated_at`,`created_at`,`has_many_id`,`id`\\) VALUES \\(\\?,\\?,\\?,\\?\\),\\(\\?,\\?,\\?,\\?\\),\\(\\?,\\?,\\?,\\?\\) ON DUPLICATE KEY UPDATE `has_many_id`=VALUES\\(`has_many_id`\\)").
		WithArgs(&now, &now, id1, id6, &now, &now, id1, id24, &now, &now, id1, id42).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectExec("DELETE FROM ones WHERE has_many_id = \\? AND id NOT IN \\(\\?,\\?,\\?\\)").WithArgs(id1, id6, id24, id42).WillReturnResult(result)

	queryResult := goSqlMock.NewRows([]string{"id", "my_test_model_id", "created_at", "updated_at"}).AddRow(*id1, nil, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND `has_manies`\\.`id` = \\? ORDER BY `has_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(queryResult)

	model := HasMany{
		Model: db_repo.Model{
			Id: id1,
		},
		Manies: []*Ones{
			{
				Model: db_repo.Model{
					Id: id6,
				},
			},
			{
				Model: db_repo.Model{
					Id: id24,
				},
			},
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

func TestRepository_UpdateHasManyNoRelation(t *testing.T) {
	now := time.Unix(1549964818, 0)
	dbc, repo := getTimedMocks(t, now, hasMany)

	result := goSqlMock.NewResult(0, 1)

	dbc.ExpectBegin()
	dbc.ExpectExec("UPDATE `has_manies` SET `updated_at`=\\?,`created_at`=\\? WHERE `id` = \\?").WithArgs(&now, &now, id1).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectExec("DELETE FROM ones WHERE has_many_id = \\?").WithArgs(id1).WillReturnResult(result)

	queryResult := goSqlMock.NewRows([]string{"id", "my_test_model_id", "created_at", "updated_at"}).AddRow(*id1, nil, now, now)
	dbc.ExpectQuery("SELECT \\* FROM `has_manies` WHERE `has_manies`\\.`id` = \\? AND `has_manies`\\.`id` = \\? ORDER BY `has_manies`\\.`id` LIMIT 1").WithArgs(*id1, *id1).WillReturnRows(queryResult)

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
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `my_test_models` WHERE `my_test_models`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies` WHERE `many_of_manies`\\.`many_to_many_id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_of_manies`  WHERE `many_of_manies`\\.`many_to_many_id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `many_to_manies` WHERE `many_to_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

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
	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `one_of_manies`  WHERE `one_of_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(result)
	dbc.ExpectCommit()

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

	dbc.ExpectExec("DELETE FROM ones WHERE has_many_id = \\?").WithArgs(*id1).WillReturnResult(childResult)

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `has_manies` WHERE `has_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(childResult)
	dbc.ExpectCommit()

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

	dbc.ExpectExec("DELETE FROM ones WHERE has_many_id = \\?").WithArgs(*id1).WillReturnResult(childResult)

	dbc.ExpectBegin()
	dbc.ExpectExec("DELETE FROM `has_manies` WHERE `has_manies`\\.`id` = \\?").WithArgs(id1).WillReturnResult(parentResult)
	dbc.ExpectCommit()

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

func TestRepository_Create_WrongModel(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	err := repo.Create(context.Background(), &ManyToMany{})

	assert.Error(t, err)
	assert.Equal(t, err, db_repo.ErrCrossCreate)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_Update_WrongModel(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	err := repo.Update(context.Background(), &ManyToMany{})

	assert.Error(t, err)
	assert.Equal(t, err, db_repo.ErrCrossUpdate)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_Delete_WrongModel(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	err := repo.Delete(context.Background(), &ManyToMany{})

	assert.Error(t, err)
	assert.Equal(t, err, db_repo.ErrCrossDelete)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_Read_WrongModel(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	err := repo.Read(context.Background(), nil, &ManyToMany{})

	assert.Error(t, err)
	assert.Equal(t, err, db_repo.ErrCrossRead)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_Query_WrongModel(t *testing.T) {
	dbc, repo := getMocks(t, myTestModel)

	err := repo.Query(context.Background(), db_repo.NewQueryBuilder(), &[]*ManyToMany{})

	assert.Error(t, err)
	assert.Equal(t, err, db_repo.ErrCrossQuery)

	if err := dbc.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func getMocks(t *testing.T, whichMetadata string) (goSqlMock.Sqlmock, db_repo.Repository) {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	clk := clock.NewFakeClock()

	db, clientMock, _ := goSqlMock.New()

	rows := goSqlMock.NewRows([]string{"SELECT VERSION()"}).AddRow("8.0.23")
	clientMock.ExpectQuery("SELECT VERSION()").WillReturnRows(rows)

	orm, err := db_repo.NewOrmWithInterfaces(clk, db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	metadata, ok := metadatas[whichMetadata]
	if !ok {
		t.Errorf("couldn't find metadata named: %s", whichMetadata)
	}

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, metadata)

	return clientMock, repo
}

func getTimedMocks(t *testing.T, time time.Time, whichMetadata string) (goSqlMock.Sqlmock, db_repo.Repository) {
	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	clk := clock.NewFakeClockAt(time)

	db, clientMock, _ := goSqlMock.New()

	rows := goSqlMock.NewRows([]string{"SELECT VERSION()"}).AddRow("8.0.23")
	clientMock.ExpectQuery("SELECT VERSION()").WillReturnRows(rows)

	orm, err := db_repo.NewOrmWithInterfaces(clk, db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	metadata, ok := metadatas[whichMetadata]
	if !ok {
		t.Errorf("couldn't find metadata named: %s", whichMetadata)
	}

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, metadata)

	return clientMock, repo
}

type SimpleStruct struct {
	Id   uint
	Name string `orm:"preload"`
}

type NestedStruct struct {
	Id              uint
	Simple          SimpleStruct `orm:"preload"`
	SimpleId        uint
	SlicedStructId  uint
	LayeredStructId uint
}

type LayeredStruct struct {
	Id            uint
	Layered       NestedStruct `orm:"preload"`
	SaladStructId uint
}

type SlicedStruct struct {
	Id     uint
	Sliced []NestedStruct `orm:"preload"`
}

type SaladStruct struct {
	Id     uint
	Sliced []LayeredStruct `orm:"preload"`
}

type NoPreload struct {
	Id     uint
	Sliced []LayeredStruct `orm:"preload:false"`
}

type SelfReference struct {
	Id   uint
	Self *SelfReference `orm:"preload"`
}

var cases = map[string]struct {
	input    interface{}
	expected []string
}{
	"struct_no_tags": {
		input:    struct{}{},
		expected: []string{},
	},
	"struct_scalar_tags": {
		// scalars don't require explicit preloading
		input:    SimpleStruct{},
		expected: []string{},
	},
	"nested_struct": {
		// one level of nesting is supported via clause.Association
		// we can add it nevertheless as it's deduplicated by GORM
		input:    NestedStruct{},
		expected: []string{"Simple"},
	},
	"layered_struct": {
		input:    LayeredStruct{},
		expected: []string{"Layered", "Layered.Simple"},
	},
	"sliced_struct": {
		input:    SlicedStruct{},
		expected: []string{"Sliced", "Sliced.Simple"},
	},
	"salad_struct": {
		input:    SaladStruct{},
		expected: []string{"Sliced", "Sliced.Layered", "Sliced.Layered.Simple"},
	},
	"self_reference": {
		input:    SelfReference{},
		expected: []string{"Self"},
	},
	"no_preload": {
		input:    NoPreload{},
		expected: []string{},
	},
}

func TestParsePreloads(t *testing.T) {
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			preloads := db_repo.ParsePreloads(c.input)

			assert.Equal(t, c.expected, preloads)
		})
	}
}
