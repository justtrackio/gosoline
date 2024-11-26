package grpcserver

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

type statsHolder struct {
	BeginTime                 time.Time
	RecvTime                  time.Time
	SentTime                  time.Time
	EndTime                   time.Time
	TotalTime                 int64
	FailFast                  bool
	IsClientStream            bool
	IsServerStream            bool
	IsTransparentRetryAttempt bool
	InHeaderWireLength        int
	InCompression             string
	FullMethod                string
	InRemoteAddr              string
	InLocalAddr               string
	InPayloadLength           int
	InHeaders                 *sync.Map
	InPayloadWireLength       int
	InPayload                 interface{}
	OutCompression            string
	OutRemoteAddr             string
	OutLocalAddr              string
	OutPayloadLength          int
	OutPayloadWireLength      int
	OutPayload                interface{}
	OutData                   []byte
	OutHeaders                *sync.Map
	Error                     error
}

func (s *statsHolder) GetLoggerFields() log.Fields {
	fields := log.Fields{
		"start_time":                   s.BeginTime,
		"recv_time":                    s.RecvTime,
		"sent_time":                    s.SentTime,
		"end_time":                     s.EndTime,
		"total_time":                   s.TotalTime,
		"fail_fast":                    s.FailFast,
		"is_client_stream":             s.IsClientStream,
		"is_server_stream":             s.IsServerStream,
		"is_transparent_retry_attempt": s.IsTransparentRetryAttempt,
		"in_header_wire_length":        s.InHeaderWireLength,
		"in_compression":               s.InCompression,
		"full_method":                  s.FullMethod,
		"in_remote_addr":               s.InRemoteAddr,
		"in_local_addr":                s.InLocalAddr,
		"in_payload_length":            s.InPayloadLength,
		"in_payload_wire_length":       s.InPayloadWireLength,
		"in_payload":                   s.InPayload,
		"out_compression":              s.OutCompression,
		"out_remote_addr":              s.OutRemoteAddr,
		"out_local_addr":               s.OutLocalAddr,
		"out_payload_length":           s.OutPayloadLength,
		"out_payload_wire_length":      s.OutPayloadWireLength,
		"out_payload":                  s.OutPayload,
		"out_data":                     s.OutData,
		"error":                        s.Error,
	}

	inHeaders := syncMapToMapStringString(s.InHeaders)
	for k, v := range inHeaders {
		fields[k] = v
	}
	outHeaders := syncMapToMapStringString(s.OutHeaders)
	for k, v := range outHeaders {
		fields[k] = v
	}

	return fields
}

func syncMapToMapStringString(sm *sync.Map) map[string]string {
	m := map[string]string{}
	sm.Range(func(kI, vI interface{}) bool {
		k, ok := kI.(string)
		if !ok {
			return false
		}

		switch v := vI.(type) {
		case string:
			m[k] = v
		case []string:
			m[k] = strings.Join(v, ",")
		case fmt.Stringer:
			m[k] = v.String()
		default:
			m[k] = fmt.Sprintf("%v", v)
		}

		return true
	})

	return m
}
