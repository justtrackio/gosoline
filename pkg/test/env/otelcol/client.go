// Package otelcol provides a client for querying telemetry data received by an
// OpenTelemetry Collector running with the debug exporter. It parses the
// collector's stdout to extract spans, metrics, and log records.
package otelcol

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Client reads and queries telemetry from the OTel collector's debug output.
type Client struct {
	containerName string
}

// NewClient creates a client that reads from the given Docker container's logs.
func NewClient(containerName string) *Client {
	return &Client{containerName: containerName}
}

// Span represents a trace span found in the collector's debug output.
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	Kind       string
	Attributes map[string]string
}

// Metric represents a metric found in the collector's debug output.
type Metric struct {
	Name        string
	Unit        string
	DataType    string
	IsMonotonic string
}

// LogRecord represents a log record found in the collector's debug output.
type LogRecord struct {
	SeverityText   string
	SeverityNumber string
	Body           string
	Attributes     map[string]string
}

// Logs returns the raw collector output.
func (c *Client) Logs() (string, error) {
	cmd := exec.Command("docker", "logs", c.containerName)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	return out.String(), nil
}

// Spans returns all spans found in the collector's debug output.
func (c *Client) Spans() ([]Span, error) {
	output, err := c.Logs()
	if err != nil {
		return nil, err
	}

	return parseSpans(output), nil
}

// Metrics returns all metrics found in the collector's debug output.
func (c *Client) Metrics() ([]Metric, error) {
	output, err := c.Logs()
	if err != nil {
		return nil, err
	}

	return parseMetrics(output), nil
}

// LogRecords returns all log records found in the collector's debug output.
func (c *Client) LogRecords() ([]LogRecord, error) {
	output, err := c.Logs()
	if err != nil {
		return nil, err
	}

	return parseLogRecords(output), nil
}

// ContainsSpan checks if a span with the given name exists in the output.
func (c *Client) ContainsSpan(name string) (bool, error) {
	spans, err := c.Spans()
	if err != nil {
		return false, err
	}

	for _, s := range spans {
		if s.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// ContainsMetric checks if a metric with the given name exists in the output.
func (c *Client) ContainsMetric(name string) (bool, error) {
	metrics, err := c.Metrics()
	if err != nil {
		return false, err
	}

	for _, m := range metrics {
		if m.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// ContainsLogRecord checks if a log record with the given body text exists in the output.
func (c *Client) ContainsLogRecord(body string) (bool, error) {
	records, err := c.LogRecords()
	if err != nil {
		return false, err
	}

	for _, r := range records {
		if strings.Contains(r.Body, body) {
			return true, nil
		}
	}

	return false, nil
}

func parseSpans(output string) []Span {
	var spans []Span
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "Span #") {
			continue
		}

		span := parseSpanBlock(lines, i+1)
		if span.Name != "" {
			spans = append(spans, span)
		}
	}

	return spans
}

func parseSpanBlock(lines []string, start int) Span {
	span := Span{Attributes: make(map[string]string)}
	inAttributes := false

	for i := start; i < min(len(lines), start+30); i++ {
		l := strings.TrimSpace(lines[i])

		if l == "" || strings.HasPrefix(l, "Span #") || strings.HasPrefix(l, "ScopeSpans") {
			break
		}

		switch {
		case strings.HasPrefix(l, "Trace ID"):
			span.TraceID = extractValue(l)
		case strings.HasPrefix(l, "Parent ID"):
			span.ParentID = extractValue(l)
		case strings.HasPrefix(l, "ID"):
			span.SpanID = extractValue(l)
		case strings.HasPrefix(l, "Name"):
			span.Name = extractValue(l)
		case strings.HasPrefix(l, "Kind"):
			span.Kind = extractValue(l)
		case l == "Attributes:":
			inAttributes = true
		case inAttributes && strings.HasPrefix(l, "-> "):
			key, val := parseAttribute(l)
			span.Attributes[key] = val
		case inAttributes && !strings.HasPrefix(l, "-> "):
			inAttributes = false
		}
	}

	return span
}

func parseMetrics(output string) []Metric {
	var metrics []Metric
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "Descriptor:" {
			continue
		}

		m := parseMetricDescriptor(lines, i+1)
		if m.Name != "" && !isInternalMetric(m.Name) {
			metrics = append(metrics, m)
		}
	}

	return metrics
}

func parseMetricDescriptor(lines []string, start int) Metric {
	var m Metric

	for i := start; i < min(len(lines), start+10); i++ {
		l := strings.TrimSpace(lines[i])

		switch {
		case strings.HasPrefix(l, "-> Name:"):
			m.Name = extractTaggedValue(l)
		case strings.HasPrefix(l, "-> Unit:"):
			m.Unit = extractTaggedValue(l)
		case strings.HasPrefix(l, "-> DataType:"):
			m.DataType = extractTaggedValue(l)
		case strings.HasPrefix(l, "-> IsMonotonic:"):
			m.IsMonotonic = extractTaggedValue(l)
		case strings.HasPrefix(l, "NumberDataPoints") || strings.HasPrefix(l, "HistogramDataPoints"):
			return m
		}
	}

	return m
}

func isInternalMetric(name string) bool {
	return strings.HasPrefix(name, "otelcol_") ||
		strings.HasPrefix(name, "scrape_") ||
		strings.HasPrefix(name, "promhttp_") ||
		name == "up"
}

func parseLogRecords(output string) []LogRecord {
	var records []LogRecord
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "Body: Str(") {
			continue
		}

		rec := parseLogRecordBlock(lines, i)
		records = append(records, rec)
	}

	return records
}

func parseLogRecordBlock(lines []string, bodyIdx int) LogRecord {
	body := lines[bodyIdx]
	body = strings.TrimSpace(body)
	body = body[len("Body: Str(") : len(body)-1]

	rec := LogRecord{Body: body, Attributes: make(map[string]string)}

	// Look backwards for severity
	for j := max(0, bodyIdx-3); j < bodyIdx; j++ {
		l := strings.TrimSpace(lines[j])
		if strings.HasPrefix(l, "SeverityText:") {
			rec.SeverityText = extractValue(l)
		}
		if strings.HasPrefix(l, "SeverityNumber:") {
			rec.SeverityNumber = extractValue(l)
		}
	}

	// Look forward for attributes
	for j := bodyIdx + 1; j < min(len(lines), bodyIdx+20); j++ {
		l := strings.TrimSpace(lines[j])
		if strings.HasPrefix(l, "-> ") {
			key, val := parseAttribute(l)
			rec.Attributes[key] = val
		} else if l == "" || strings.HasPrefix(l, "Trace ID") || strings.HasPrefix(l, "LogRecord #") {
			break
		}
	}

	return rec
}

func extractValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func extractTaggedValue(line string) string {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return ""
	}

	return strings.TrimSpace(line[idx+1:])
}

func parseAttribute(line string) (key, val string) {
	// "     -> foo: Str(bar)" -> "foo", "bar"
	line = strings.TrimPrefix(line, "-> ")
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}

	key = strings.TrimSpace(parts[0])
	val = strings.TrimSpace(parts[1])

	// Strip type wrapper: Str(bar) -> bar, Int(42) -> 42
	if idx := strings.Index(val, "("); idx >= 0 {
		val = val[idx+1:]
		if end := strings.LastIndex(val, ")"); end >= 0 {
			val = val[:end]
		}
	}

	return key, val
}
