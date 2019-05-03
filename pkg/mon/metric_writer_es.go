package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/es"
	"github.com/jonboulle/clockwork"
	"github.com/olivere/elastic/v7"
)

type esMetricDatum struct {
	*MetricDatum
	Namespace string `json:"namespace"`
}

type esWriter struct {
	logger    Logger
	clock     clockwork.Clock
	client    *elastic.Client
	namespace string
}

func NewMetricEsWriter(config cfg.Config, logger Logger) *esWriter {
	client := es.ProvideClient(config, logger, "metric")
	clock := clockwork.NewRealClock()

	appId := cfg.GetAppIdFromConfig(config)
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	return NewMetricEsWriterWithInterfaces(logger, client, clock, namespace)
}

func NewMetricEsWriterWithInterfaces(logger Logger, client *elastic.Client, clock clockwork.Clock, namespace string) *esWriter {
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

func (w esWriter) Write(batch MetricData) {
	svc := elastic.NewBulkService(w.client)

	for i := range batch {
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}

		m := esMetricDatum{
			MetricDatum: batch[i],
			Namespace:   w.namespace,
		}

		index := fmt.Sprintf("metrics-%s", m.Timestamp.Format("20060102"))

		req := elastic.NewBulkIndexRequest().
			Index(index).
			Doc(m)

		svc.Add(req)
	}

	_, err := svc.Do(context.TODO())

	if err != nil {
		w.logger.Error(err, "could not write metric data to es")
		return
	}

	w.logger.Debugf("written %d metric data sets to elasticsearch", len(batch))
}

func (w esWriter) WriteOne(data *MetricDatum) {
	w.Write(MetricData{data})
}
