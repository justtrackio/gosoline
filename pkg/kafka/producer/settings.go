package producer

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaCompressionCodec string

const (
	CompressionNone   KafkaCompressionCodec = "none"
	CompressionGZip   KafkaCompressionCodec = "gzip"
	CompressionSnappy KafkaCompressionCodec = "snappy"
	CompressionLZ4    KafkaCompressionCodec = "lz4"
	CompressionZstd   KafkaCompressionCodec = "zstd"
)

type Settings struct {
	Identity       cfg.Identity `cfg:"identity"`
	Connection     string
	TopicId        string
	Compression    KafkaCompressionCodec
	MaxBatchBytes  int32
	MaxBatchSize   int
	LingerTimeout  time.Duration
	RequestTimeout time.Duration
}

func (s Settings) GetKafkaCompressor() kgo.CompressionCodec {
	switch s.Compression {
	case CompressionGZip:
		return kgo.GzipCompression()
	case CompressionSnappy:
		return kgo.SnappyCompression()
	case CompressionLZ4:
		return kgo.Lz4Compression()
	case CompressionZstd:
		return kgo.ZstdCompression()
	default:
		return kgo.NoCompression()
	}
}
