package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/indexed-store"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
)

// TODO: make a real test out of this

func main() {
	app := application.Default()
	app.Add("test", newTestModule)
	app.Run()
}

type testModule struct {
	store indexed_store.IndexedStore
}

type TestModel struct {
	Id      uint   `json:"id" ddb:"key=hash"`
	Name    string `json:"name"`
	Package string `json:"package"`
	Data    string `json:"data"`
}

func (m TestModel) GetId() interface{} {
	return m.Id
}

type TestModelByName struct {
	Id      uint   `json:"id"`
	Name    string `json:"name" ddb:"global=hash"`
	Package string `json:"package"`
	Data    string `json:"data"`
}

func (m TestModelByName) ToBaseValue() indexed_store.BaseValue {
	return TestModel(m)
}

type TestModelByPackageAndName struct {
	Id      uint   `json:"id"`
	Name    string `json:"name" ddb:"global=range"`
	Package string `json:"package" ddb:"global=hash"`
	Data    string `json:"data"`
}

func (m TestModelByPackageAndName) ToBaseValue() indexed_store.BaseValue {
	return TestModel(m)
}

func newTestModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	store, err := indexed_store.NewDdbIndexedStore(config, logger, &indexed_store.Settings{
		Name:  "testModel",
		Model: TestModel{},
		Indices: []indexed_store.IndexSettings{
			{
				Name:  "IDX_by_name",
				Model: TestModelByName{},
			},
			{
				Name:  "IDX_by_package_and_name",
				Model: TestModelByPackageAndName{},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &testModule{
		store: store,
	}, nil
}

func (m testModule) Run(ctx context.Context) error {
	err := m.store.Put(ctx, TestModel{
		Id:      1,
		Name:    "Titanic",
		Package: "Romance",
		Data:    "Crash into an iceberg!",
	})
	if err != nil {
		return err
	}

	err = m.store.Put(ctx, TestModel{
		Id:      2,
		Name:    "Casablanca",
		Package: "Romance",
		Data:    "Something with an airport",
	})
	if err != nil {
		return err
	}

	err = m.store.Put(ctx, TestModel{
		Id:      3,
		Name:    "Rambo",
		Package: "Action",
		Data:    "Shoot a lot of bullets",
	})
	if err != nil {
		return err
	}

	item, err := m.store.Get(ctx, 2)
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.Get(ctx, 1)
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.Get(ctx, 3)
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_name", "Casablanca")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_name", "Rambo")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_name", "Titanic")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_package_and_name", "Romance", "Casablanca")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_package_and_name", "Action", "Rambo")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	item, err = m.store.GetFromIndex(ctx, "IDX_by_package_and_name", "Romance", "Titanic")
	if err != nil {
		return err
	}
	fmt.Println(item.(TestModel))

	return nil
}
