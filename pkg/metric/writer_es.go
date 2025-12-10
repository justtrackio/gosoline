package metric

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/es"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	RegisterWriterFactory(WriterTypeElasticsearch, ProvideElasticsearchWriter)
}

var _ Writer = &elasticsearchWriter{}

type esMetricDatum struct {
	*Datum
	Namespace string `json:"namespace"`
}

type ElasticsearchSettings struct {
	WriteGraceTime time.Duration `cfg:"write_grace_time" default:"10s"`
}

type elasticsearchWriter struct {
	logger         log.Logger
	clock          clock.Clock
	client         *es.ClientV7
	namespace      string
	writeGraceTime time.Duration
}

type esWriterCtxKey string

func ProvideElasticsearchWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, esWriterCtxKey("default"), func() (Writer, error) {
		return NewElasticsearchWriter(ctx, config, logger)
	})
}

func NewElasticsearchWriter(_ context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	esSettings := &ElasticsearchSettings{}
	if err := getMetricWriterSettings(config, WriterTypeElasticsearch, esSettings); err != nil {
		return nil, fmt.Errorf("could not get elasticsearch writer settings: %w", err)
	}

	client, err := es.ProvideClient(config, es.NewLogger(logger), "metric")
	if err != nil {
		return nil, fmt.Errorf("can not create es client: %w", err)
	}

	testClock := clock.NewRealClock()

	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not get app id from config: %w", err)
	}
	namespace := fmt.Sprintf("%s/%s/%s/%s-%s", appId.Project, appId.Environment, appId.Family, appId.Group, appId.Application)

	return NewElasticsearchWriterWithInterfaces(logger, client, testClock, namespace, esSettings.WriteGraceTime), nil
}

func NewElasticsearchWriterWithInterfaces(
	logger log.Logger,
	client *es.ClientV7,
	clock clock.Clock,
	namespace string,
	writeGraceTime time.Duration,
) Writer {
	return &elasticsearchWriter{
		logger:         logger.WithChannel("metrics"),
		clock:          clock,
		client:         client,
		namespace:      namespace,
		writeGraceTime: writeGraceTime,
	}
}

func (w elasticsearchWriter) GetPriority() int {
	return PriorityLow
}

func (w elasticsearchWriter) bulkWriteToES(ctx context.Context, buf bytes.Buffer) {
	batchReader := bytes.NewReader(buf.Bytes())

	res, err := w.client.Bulk(batchReader)
	if err != nil {
		w.logger.Error(ctx, "could not write metric data to es: %w", err)

		return
	}

	if res.IsError() {
		// A successful response might still contain errors for particular documents
		w.logger.WithFields(log.Fields{
			"status_code": res.StatusCode,
		}).Error(ctx, "not all metrics have been written to es: %w", err)
	}
}

func (w elasticsearchWriter) Write(applicationCtx context.Context, batch Data) {
	if len(batch) == 0 {
		return
	}

	delayedCtx, stop := exec.WithDelayedCancelContext(applicationCtx, w.writeGraceTime)
	defer stop()

	w.write(delayedCtx, batch)
}

func (w elasticsearchWriter) write(ctx context.Context, batch Data) {
	var buf bytes.Buffer

	for i := range batch {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}

		m := esMetricDatum{
			Datum:     batch[i],
			Namespace: w.namespace,
		}

		data, err := json.Marshal(m)
		if err != nil {
			w.logger.Error(ctx, "could not marshal metric data and write to es: %w", err)

			continue
		}

		index := m.Timestamp.Format("20060102")

		buf.Write([]byte(
			fmt.Sprintf(`{ "index" : { "_index" : "metrics-%s", "_type" : "_doc" } }%s`, index, "\n"),
		))

		buf.Write(data)
		buf.Write([]byte{10})
	}

	w.bulkWriteToES(ctx, buf)

	w.logger.Debug(ctx, "written %d metric data sets to elasticsearch", len(batch))
}

func (w elasticsearchWriter) WriteOne(ctx context.Context, data *Datum) {
	w.Write(ctx, Data{data})
}
