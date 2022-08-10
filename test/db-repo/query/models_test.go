//go:build integration

package query_test

import (
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TestModel struct {
	db_repo.Model
	Name *string
}

var TestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel",
	},
	TableName:  "test_models",
	PrimaryKey: "test_models.id",
	Mappings: db_repo.FieldMappings{
		"testModel.id":   db_repo.NewFieldMapping("test_models.id"),
		"testModel.name": db_repo.NewFieldMapping("test_models.name"),
	},
}

type TestMany struct {
	db_repo.Model
	Name   string
	Others []*TestManyToMany `gorm:"many2many:test_many_to_manies;joinReferences:manyId" orm:"preload"`
}

type TestManyToMany struct {
	db_repo.Model
	ManyId  *uint
	Other   *TestMany `gorm:"joinForeignKey:otherId" orm:"preload"`
	OtherId *uint
}

var TestManyMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testMany",
	},
	TableName:  "test_manies",
	PrimaryKey: "test_manies.id",
	Mappings: db_repo.FieldMappings{
		"testMany.id":   db_repo.NewFieldMapping("test_manies.id"),
		"testMany.name": db_repo.NewFieldMapping("test_manies.name"),
	},
	Preloads: db_repo.ParsePreloads(TestMany{}),
}
