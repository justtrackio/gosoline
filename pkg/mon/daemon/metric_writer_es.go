package daemon

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
)

type esMetricDatum struct {
	*mon.MetricDatum
	Namespace string `json:"namespace"`
}

type esWriter struct {
	logger    mon.Logger
	clock     clockwork.Clock
	client    *es.ClientV7
	namespace string
}

func NewMetricEsWriter(config cfg.Config, logger mon.Logger) *esWriter {
	client := es.ProvideClient(config, logger, "metric")
	clock := clockwork.NewRealClock()

	appId := cfg.GetAppIdFromConfig(config)
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	return NewMetricEsWriterWithInterfaces(logger, client, clock, namespace)
}

func NewMetricEsWriterWithInterfaces(logger mon.Logger, client *es.ClientV7, clock clockwork.Clock, namespace string) *esWriter {
	return &esWriter{
		logger:    logger.WithChannel("metrics"),
		clock:     clock,
		client:    client,
		namespace: namespace,
	}
}

func (w esWriter) GetPriority() int {
	return mon.PriorityLow
}

func (w esWriter) bulkWriteToES(buf bytes.Buffer) {
	batchReader := bytes.NewReader(buf.Bytes())

	res, err := w.client.Bulk(batchReader)
	if err != nil {
		w.logger.Error(err, "could not write metric data to es")
		return
	}

	if res.IsError() {
		// A successful response might still contain errors for particular documents
		w.logger.WithFields(mon.Fields{
			"status_code": res.StatusCode,
		}).Error(err, "not all metrics have been written to es")
	}
}

func (w esWriter) Write(batch mon.MetricData) {
	var buf bytes.Buffer

	if len(batch) == 0 {
		return
	}

	for i := range batch {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}

		m := esMetricDatum{
			MetricDatum: batch[i],
			Namespace:   w.namespace,
		}

		data, err := json.Marshal(m)
		if err != nil {
			w.logger.Error(err, "could not marshal metric data and write to es")
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

	w.logger.Debugf("written %d metric data sets to elasticsearch", len(batch))
}

func (w esWriter) WriteOne(data *mon.MetricDatum) {
	w.Write(mon.MetricData{data})
}
