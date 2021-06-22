package metric

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/log"
	"github.com/jonboulle/clockwork"
)

type esMetricDatum struct {
	*Datum
	Namespace string `json:"namespace"`
}

type esWriter struct {
	logger    log.Logger
	clock     clockwork.Clock
	client    *es.ClientV7
	namespace string
}

func NewEsWriter(config cfg.Config, logger log.Logger) (*esWriter, error) {
	client, err := es.ProvideClient(config, logger, "metric")
	if err != nil {
		return nil, fmt.Errorf("can not create es client: %w", err)
	}

	clock := clockwork.NewRealClock()

	appId := cfg.GetAppIdFromConfig(config)
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	return NewEsWriterWithInterfaces(logger, client, clock, namespace), nil
}

func NewEsWriterWithInterfaces(logger log.Logger, client *es.ClientV7, clock clockwork.Clock, namespace string) *esWriter {
	return &esWriter{
		logger:    logger.WithChannel("metrics"),
		clock:     clock,
		client:    client,
		namespace: namespace,
	}
}

func (w esWriter) GetPriority() int {
	return PriorityLow
}

func (w esWriter) bulkWriteToES(buf bytes.Buffer) {
	batchReader := bytes.NewReader(buf.Bytes())

	res, err := w.client.Bulk(batchReader)
	if err != nil {
		w.logger.Error("could not write metric data to es: %w", err)
		return
	}

	if res.IsError() {
		// A successful response might still contain errors for particular documents
		w.logger.WithFields(log.Fields{
			"status_code": res.StatusCode,
		}).Error("not all metrics have been written to es: %w", err)
	}
}

func (w esWriter) Write(batch Data) {
	var buf bytes.Buffer

	if len(batch) == 0 {
		return
	}

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
			w.logger.Error("could not marshal metric data and write to es: %w", err)
			continue
		}

		index := m.Timestamp.Format("20060102")

		buf.Write([]byte(
			fmt.Sprintf(`{ "index" : { "_index" : "metrics-%s", "_type" : "_doc" } }%s`, index, "\n"),
		))

		buf.Write(data)
		buf.Write([]byte{10})
	}

	w.bulkWriteToES(buf)

	w.logger.Debug("written %d metric data sets to elasticsearch", len(batch))
}

func (w esWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}
