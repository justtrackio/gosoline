package parquet

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/reader"
	"golang.org/x/sync/semaphore"
	"strings"
	"sync"
	"time"
)

type Progress struct {
	FileCount int
	Current   int
}

type ReadResult map[string]interface{}
type ReadResults []ReadResult
type ResultCallback func(progress Progress, result ReadResults) (bool, error)

//go:generate mockery -name Reader
type Reader interface {
	ReadDate(ctx context.Context, datetime time.Time) (ReadResults, error)
	ReadDateAsync(ctx context.Context, datetime time.Time, callback ResultCallback) error
	ReadFile(ctx context.Context, file string) (ReadResults, error)
}

type s3Reader struct {
	logger   mon.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	prefixNamingStrategy s3PrefixNamingStrategy

	settings *Settings
}

func NewReader(config cfg.Config, logger mon.Logger, settings *Settings) *s3Reader {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		panic(fmt.Sprintf("Unknown prefix naming strategy '%s'", settings.NamingStrategy))
	}

	return NewReaderWithInterfaces(logger, s3Cfg, s3Client, prefixNaming, settings)
}

func NewReaderWithInterfaces(logger mon.Logger, s3Cfg *aws.Config, s3Client s3iface.S3API, prefixNaming s3PrefixNamingStrategy, settings *Settings) *s3Reader {
	return &s3Reader{
		logger:               logger,
		s3Cfg:                s3Cfg,
		s3Client:             s3Client,
		prefixNamingStrategy: prefixNaming,
		settings:             settings,
	}
}

func (r *s3Reader) ReadDate(ctx context.Context, datetime time.Time) (ReadResults, error) {
	tmp := make(map[int]ReadResults)
	err := r.ReadDateAsync(ctx, datetime, func(progress Progress, result ReadResults) (bool, error) {
		tmp[progress.Current] = result

		return true, nil
	})

	if err != nil {
		return nil, err
	}

	result := make(ReadResults, 0)

	for _, partial := range tmp {
		result = append(result, partial...)
	}

	return result, nil
}

func (r *s3Reader) ReadDateAsync(ctx context.Context, datetime time.Time, callback ResultCallback) error {
	prefix := r.prefixNamingStrategy(r.settings.ModelId, datetime)
	files, err := r.listFiles(prefix)

	if err != nil {
		return err
	}

	fileCount := len(files)
	stop := false

	sem := semaphore.NewWeighted(int64(10))
	wg := sync.WaitGroup{}
	wg.Add(fileCount)

	for i, file := range files {
		err := sem.Acquire(ctx, int64(1))

		if err != nil {
			return err
		}

		go func(i int, file string) {
			defer sem.Release(int64(1))
			defer wg.Done()

			if stop {
				return
			}

			r.logger.Debugf("reading file %d of %d: %s", i, fileCount, file)

			result, err := r.ReadFile(ctx, file)

			if err != nil {
				r.logger.Fatal(err, "can not read file")
				return
			}

			ok, err := callback(Progress{
				FileCount: fileCount,
				Current:   i,
			}, result)

			if !ok {
				stop = true
				return
			}
		}(i, file)
	}

	wg.Wait()

	return nil
}

func (r *s3Reader) ReadFile(ctx context.Context, file string) (ReadResults, error) {
	bucket := r.getBucketName()
	fr, err := parquetS3.NewS3FileReader(ctx, bucket, file, r.s3Cfg)

	if err != nil {
		return nil, err
	}

	pr, err := reader.NewParquetColumnReader(fr, 4)

	if err != nil {
		return nil, err
	}

	size := int(pr.GetNumRows())
	columnNames := pr.SchemaHandler.ValueColumns
	columns := make(map[string][]interface{})
	rootName := pr.Footer.Schema[0].Name

	for _, colSchema := range columnNames {
		values, _, _, err := pr.ReadColumnByPath(colSchema, size)

		if err != nil {
			return nil, err
		}

		key := strings.ToLower(colSchema[len(rootName)+1:])

		columns[key] = values
	}

	pr.ReadStop()
	if err = fr.Close(); err != nil {
		return nil, err
	}

	results := make([]ReadResult, size)

	for key, values := range columns {
		for i := 0; i < size; i++ {
			if results[i] == nil {
				results[i] = make(ReadResult, len(columns))
			}

			results[i][key] = values[i]
		}
	}

	return results, nil
}

func (r *s3Reader) listFiles(prefix string) ([]string, error) {
	bucket := r.getBucketName()

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	files := make([]string, 0, 128)

	for {
		out, err := r.s3Client.ListObjects(input)

		if err != nil {
			return nil, err
		}

		if len(out.Contents) == 0 {
			break
		}

		for _, obj := range out.Contents {
			files = append(files, *obj.Key)
		}

		if !*out.IsTruncated {
			break
		}

		input.Marker = out.Contents[len(out.Contents)-1].Key
	}

	return files, nil
}

func (r *s3Reader) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     r.settings.ModelId.Project,
		Environment: r.settings.ModelId.Environment,
		Family:      r.settings.ModelId.Family,
		Application: r.settings.ModelId.Application,
	})
}

func (r ReadResult) GetString(key string) string {
	value, ok := r[key]

	if !ok {
		return ""
	}

	str, ok := value.(string)

	if !ok {
		return ""
	}

	return str
}

func (r ReadResult) GetInt32(key string) int32 {
	value, ok := r[key]

	if !ok || value == nil {
		return 0
	}

	i, ok := value.(int32)

	if !ok {
		return 0
	}

	return i
}

func (r ReadResult) GetTime(key string) time.Time {
	value, ok := r[key]
	if !ok {
		return time.Time{}
	}

	integer, ok := value.(int64)

	if !ok {
		return time.Time{}
	}

	return time.Unix(0, integer*int64(time.Millisecond))
}

func (r ReadResult) GetFloat32(key string) float32 {
	value, ok := r[key]

	if !ok || value == nil {
		return 0
	}

	i, ok := value.(float32)

	if !ok {
		return 0
	}

	return i
}

func (r ReadResult) GetBool(key string) bool {
	value, ok := r[key]

	if !ok || value == nil {
		return false
	}

	i, ok := value.(bool)

	if !ok {
		return false
	}

	return i
}

func (r ReadResult) GetUint(key string) uint {
	value, ok := r[key]

	if !ok || value == nil {
		return 0
	}

	i, ok := value.(int32)

	if !ok {
		return 0
	}

	return uint(i)
}

func (r ReadResult) GetFloat64(key string) float64 {
	value, ok := r[key]

	if !ok || value == nil {
		return 0
	}

	i, ok := value.(float32)

	if !ok {
		return 0
	}

	return float64(i)
}
