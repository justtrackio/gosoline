package mdlsub

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type FixtureSettings struct {
	Dataset FixtureSettingsDataset `cfg:"dataset"`
	Host    string                 `cfg:"host"`
	Path    string                 `cfg:"path"`
}

type FixtureSettingsDataset struct {
	Id int `cfg:"id"`
}

type fetchResult struct {
	spec *ModelSpecification
	data *FetchData
}

type FetchData struct {
	Data json.RawMessage `json:"data"`
}

func FixtureSetFactory(transformerFactoryMap TransformerMapTypeVersionFactories) fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		var err error
		var core SubscriberCore
		var sets []fixtures.FixtureSet

		settings := unmarshalSettings(config)

		key := fmt.Sprintf("fixtures.providers.%s", group)
		fixtureSettings := &FixtureSettings{}
		config.UnmarshalKey(key, fixtureSettings)

		if core, err = NewSubscriberCore(ctx, config, logger, settings.Subscribers, transformerFactoryMap); err != nil {
			return nil, fmt.Errorf("failed to create subscriber core: %w", err)
		}

		for _, sub := range settings.Subscribers {
			sets = append(sets, NewFixtureSet(logger, sub.SourceModel, core, fixtureSettings))
		}

		return sets, nil
	}
}

type FixtureSet struct {
	logger     log.Logger
	source     SubscriberModel
	core       SubscriberCore
	httpClient *resty.Client
	settings   *FixtureSettings
}

func NewFixtureSet(logger log.Logger, source SubscriberModel, core SubscriberCore, settings *FixtureSettings) *FixtureSet {
	return NewFixtureSetWithInterfaces(logger, source, core, settings, resty.New())
}

func NewFixtureSetWithInterfaces(logger log.Logger, source SubscriberModel, core SubscriberCore, settings *FixtureSettings, httpClient *resty.Client) *FixtureSet {
	logger = logger.WithChannel("fixtures").WithFields(map[string]any{
		"model_id": source.String(),
	})

	return &FixtureSet{
		logger:     logger.WithChannel("fixtures"),
		source:     source,
		core:       core,
		httpClient: httpClient,
		settings:   settings,
	}
}

func (f FixtureSet) Write(ctx context.Context) error {
	var err error
	var res *fetchResult
	var transformer ModelTransformer
	var output Output
	var items []any
	var model Model

	if res, err = f.fetch(ctx); err != nil {
		return fmt.Errorf("failed to fetch model %s: %w", f.source.String(), err)
	}

	if transformer, err = f.core.GetTransformer(res.spec); err != nil {
		return fmt.Errorf("failed to get transformer: %w", err)
	}

	input := transformer.GetInput()
	slice := refl.CreatePointerToSliceOfTypeAndSize(input, 0)

	if err = json.Unmarshal(res.data.Data, slice); err != nil {
		return fmt.Errorf("failed to unmarshal fetched data: %w", err)
	}

	if items, err = refl.InterfaceToInterfaceSlice(slice); err != nil {
		return fmt.Errorf("failed to interface slice: %w", err)
	}

	if output, err = f.core.GetOutput(res.spec); err != nil {
		return fmt.Errorf("failed to get output: %w", err)
	}

	for _, item := range items {
		mdl := refl.ValueToPointerValue(item)

		if model, err = transformer.Transform(ctx, mdl); err != nil {
			return fmt.Errorf("failed to transform model %s: %w", f.source.String(), err)
		}

		if err = output.Persist(ctx, model, res.spec.CrudType); err != nil {
			return fmt.Errorf("failed to persist model %s: %w", f.source.String(), err)
		}
	}

	f.logger.WithContext(ctx).Info("persisted %d fixtures", len(items))

	return nil
}

func (f FixtureSet) fetch(ctx context.Context) (*fetchResult, error) {
	var err error
	var hostPath string
	var u *url.URL
	var resp *resty.Response
	var version int

	if hostPath, err = url.JoinPath(f.settings.Host, f.settings.Path); err != nil {
		return nil, fmt.Errorf("failed to join host path: %w", err)
	}

	if u, err = url.Parse(hostPath); err != nil {
		return nil, fmt.Errorf("failed to parse host url %s: %w", hostPath, err)
	}

	if version, err = f.core.GetLatestModelIdVersion(f.source.ModelId); err != nil {
		return nil, fmt.Errorf("failed to get latest model version: %w", err)
	}

	query := u.Query()
	query.Add("dataset_id", strconv.Itoa(f.settings.Dataset.Id))
	query.Add("model_id", f.source.String())
	query.Add("version", strconv.Itoa(version))
	u.RawQuery = query.Encode()

	data := &FetchData{}
	req := f.httpClient.R().SetContext(ctx).SetResult(data)
	f.logger.WithContext(ctx).Info("fetching fixture data from %s", u.String())

	if resp, err = req.Get(u.String()); err != nil {
		return nil, fmt.Errorf("error on executing http request data: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("error on executing http request data, got status %d: %s", resp.StatusCode(), resp.String())
	}

	res := &fetchResult{
		spec: &ModelSpecification{
			ModelId:  f.source.String(),
			CrudType: "create",
			Version:  version,
		},
		data: data,
	}

	return res, nil
}
