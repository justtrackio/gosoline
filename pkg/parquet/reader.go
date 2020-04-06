package parquet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	parquetBuffer "github.com/xitongsys/parquet-go-source/buffer"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/source"
	"golang.org/x/sync/semaphore"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

type Progress struct {
	FileCount    int
	BytesCount   int64
	Current      int
	CurrentBytes int64
	LastChunk    bool
}

type ReadResult map[string]interface{}
type ReadResults []ReadResult
type ResultCallback func(progress Progress, results interface{}) (bool, error)

var CallbackStopErr = errors.New("callback returned false")

//go:generate mockery -name Reader
type Reader interface {
	ReadDates(ctx context.Context, dates []time.Time, target interface{}) error
	ReadDatesAsync(ctx context.Context, dates []time.Time, target interface{}, callback ResultCallback) error
	ReadFile(ctx context.Context, file string, callback func(chunk ReadResults, lastChunk bool) error) error
}

type ReaderOption func(s *s3Reader) error

type s3Reader struct {
	logger   mon.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	modelId              mdl.ModelId
	prefixNamingStrategy s3PrefixNamingStrategy
	recorder             FileRecorder
	batchSize            int
	maxConcurrentRows    int64
	cacheDirectory       *string
}

func NewReader(config cfg.Config, logger mon.Logger, opts ...ReaderOption) *s3Reader {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)

	s := NewReaderWithInterfaces(
		logger,
		s3Cfg,
		s3Client,
		mdl.ModelId{},
		s3PrefixNamingStrategies[NamingStrategyDtSeparated],
		NewNopRecorder(),
		10_000,
		1_000_000,
		nil,
	)

	for _, opt := range opts {
		if err := opt(s); err != nil {
			logger.Panic(err, "failed to configure S3 parquet reader")
		}
	}

	s.modelId.PadFromConfig(config)
	if len(s.modelId.Name) == 0 {
		err := errors.New("model may not be empty")
		logger.Panic(err, "failed to configure S3 parquet reader")
	}

	return s
}

func NewReaderWithInterfaces(
	logger mon.Logger,
	s3Cfg *aws.Config,
	s3Client s3iface.S3API,
	modelId mdl.ModelId,
	prefixNaming s3PrefixNamingStrategy,
	recorder FileRecorder,
	batchSize int,
	maxConcurrentRows int64,
	cacheDirectory *string,
) *s3Reader {
	return &s3Reader{
		logger:               logger,
		s3Cfg:                s3Cfg,
		s3Client:             s3Client,
		modelId:              modelId,
		prefixNamingStrategy: prefixNaming,
		recorder:             recorder,
		batchSize:            batchSize,
		maxConcurrentRows:    maxConcurrentRows,
		cacheDirectory:       cacheDirectory,
	}
}

func ReaderModelId(modelId mdl.ModelId) ReaderOption {
	return func(s *s3Reader) error {
		s.modelId = modelId

		return nil
	}
}

func ReaderNamingStrategy(namingStrategy string) ReaderOption {
	return func(s *s3Reader) error {
		prefixNaming, exists := s3PrefixNamingStrategies[namingStrategy]

		if !exists {
			return fmt.Errorf("unknown prefix naming strategy '%s'", namingStrategy)
		}

		s.prefixNamingStrategy = prefixNaming

		return nil
	}
}

func ReaderFileRecorder(recorder FileRecorder) ReaderOption {
	return func(s *s3Reader) error {
		if recorder == nil {
			recorder = NewNopRecorder()
		}

		s.recorder = recorder

		return nil
	}
}

// how many rows do you want to receive at once in your callback?
// if e.g. the file you are reading has 400k rows (could be around
// 30mb) and we instantiate all of them at once, we create quite a
// lot of objects at the same time, causing memory explosion (10gb)
// instead, if we only parse 400k rows and then convert them in
// batches of 10k to your application type, we can reduce memory
// usage quite a bit
// default: 10 000
func ReaderBatchSize(batchSize int) ReaderOption {
	return func(s *s3Reader) error {
		if batchSize <= 0 {
			return fmt.Errorf("batch size needs to be positive")
		}

		s.batchSize = batchSize

		return nil
	}
}

// how many rows should we try to keep in memory at once? if multiple
// workers are processing big files, you can easily multiply your memory
// usage by the number of your workers. instead, each worker now
// has to keep track of the number of rows it is processing
// default: 1 000 000, should result in 2-3gb memory usage
func ReaderMaxConcurrentRows(maxConcurrentRows int64) ReaderOption {
	return func(s *s3Reader) error {
		if maxConcurrentRows <= 0 {
			return fmt.Errorf("max concurrent rows needs to be positive")
		}

		s.maxConcurrentRows = maxConcurrentRows

		return nil
	}
}

// allow the reader to store downloaded files in some directory on your hard disk.
// if you process the same files multiple times this will allow you to skip downloading
// them again and again.
// you should have a few gb to spare in this directory
// default: do not cache downloaded files
func ReaderCacheDirectory(cacheDirectory string) ReaderOption {
	return func(s *s3Reader) error {
		if len(cacheDirectory) == 0 {
			return fmt.Errorf("can not set cache directory to empty string")
		}

		s.cacheDirectory = &cacheDirectory

		return nil
	}
}

func (r *s3Reader) ReadDates(ctx context.Context, dates []time.Time, target interface{}) error {
	if !refl.IsPointerToSlice(target) {
		return fmt.Errorf("target needs to be a pointer to a slice, but is %T", target)
	}

	tmp := refl.CreatePointerToSliceOfTypeAndSize(target, 0)
	tp := reflect.ValueOf(tmp)
	t := tp.Elem()

	err := r.ReadDatesAsync(ctx, dates, target, func(progress Progress, result interface{}) (bool, error) {
		rp := reflect.ValueOf(result)
		r := rp.Elem()

		t.Set(reflect.AppendSlice(t, r))

		return true, nil
	})

	if err != nil {
		return err
	}

	refl.CopyPointerSlice(target, tmp)

	return nil
}

func (r *s3Reader) ReadDatesAsync(ctx context.Context, dates []time.Time, target interface{}, callback ResultCallback) error {
	if !refl.IsPointerToSlice(target) {
		return fmt.Errorf("target needs to be a pointer to a slice, but is %T", target)
	}

	files := make([]string, 0, len(dates))
	var totalSize int64

	for _, date := range dates {
		newFiles, newSize, err := r.listFilesFromDate(date)

		if err != nil {
			return err
		}

		files = append(files, newFiles...)
		totalSize += newSize
	}

	return r.readFilesAsync(ctx, files, totalSize, target, callback)
}

func (r *s3Reader) readFilesAsync(ctx context.Context, files []string, totalSize int64, target interface{}, callback ResultCallback) error {
	fileCount := len(files)

	// amount of concurrent workers who can download new files
	dlSem := semaphore.NewWeighted(int64(runtime.NumCPU() * 2))
	// amount of concurrent workers processing the files
	cbSem := semaphore.NewWeighted(int64(runtime.NumCPU()))
	// amount of rows we want to process at once (avoids too high memory pressure)
	rowsSem := semaphore.NewWeighted(r.maxConcurrentRows)
	cfn, cfnCtx := coffin.WithContext(ctx)
	var progress int32
	var progressBytes int64

	for _, file := range files {
		file := file
		cfn.GoWithContext(cfnCtx, func(ctx context.Context) error {
			if err := dlSem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer dlSem.Release(1)

			logger := r.logger.WithContext(ctx)
			logger.Infof("reading file %s", file)

			size, err := r.readFile(ctx, file, func(result ReadResults, last bool) error {
				if err := cbSem.Acquire(ctx, 1); err != nil {
					return err
				}
				defer cbSem.Release(1)

				decoded := refl.CreatePointerToSliceOfTypeAndSize(target, len(result))

				if err := r.decode(result, decoded); err != nil {
					return fmt.Errorf("can not decode results for file %s: %w", file, err)
				}

				ok, err := callback(Progress{
					FileCount:    fileCount,
					BytesCount:   totalSize,
					Current:      int(progress),
					CurrentBytes: progressBytes,
					LastChunk:    last,
				}, decoded)

				if !ok {
					cfn.Kill(CallbackStopErr)
				}

				return err
			}, rowsSem, r.maxConcurrentRows)

			atomic.AddInt32(&progress, 1)
			atomic.AddInt64(&progressBytes, size)
			logger.Infof("processed file: %s", file)
			r.recorder.RecordFile(r.getBucketName(), file)

			if err != nil {
				return fmt.Errorf("can not process file %s: %w", file, err)
			}

			return nil
		})
	}

	return cfn.Wait()
}

func (r *s3Reader) downloadFile(bucket string, file string) ([]byte, error) {
	if r.cacheDirectory == nil {
		return r.downloadFileFromS3(bucket, file)
	}

	byteKey := sha256.Sum256([]byte(fmt.Sprintf("%s/%s", bucket, file)))
	key := hex.EncodeToString(byteKey[:])
	fullPath := fmt.Sprintf("%s/%s", *r.cacheDirectory, key)

	// we don't really care why we could not read the file, most likely it did not exist
	// but even if something else was the reason, we just download a fresh copy and are done
	if contents, err := ioutil.ReadFile(fullPath); err == nil {
		return contents, nil
	}

	contents, err := r.downloadFileFromS3(bucket, file)

	if err != nil {
		return nil, err
	}

	if err := os.Mkdir(*r.cacheDirectory, 0755); !os.IsExist(err) {
		// we can not write to the cache, so lets just ignore the cache
		return contents, nil
	}

	// write to a temp file first - if we get interrupted while writing (crash in another thread, SIGTERM),
	// we will have only a partial file written. Instead, if we write to another file first, we only corrupt
	// that file, but no one will ever read it
	tmpPath := fmt.Sprintf("%s.tmp", fullPath)

	// again we don't really care about any errors - if we fail to write to the cache, so be it
	// however, if we e.g. run out of space, we want to make sure we are not leaving behind partial
	// files, so we better delete the file again
	if err := ioutil.WriteFile(tmpPath, contents, 0644); err != nil {
		_ = os.Remove(tmpPath)

		return contents, nil
	}

	// try to make it official. if it fails, just clean everything and try again next time
	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		_ = os.Remove(fullPath)
	}

	return contents, nil
}

func (r *s3Reader) downloadFileFromS3(bucket string, file string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	}

	out, err := r.s3Client.GetObject(input)

	if err != nil {
		return nil, err
	}

	contents, readErr := ioutil.ReadAll(out.Body)

	if err := out.Body.Close(); err != nil {
		return nil, err
	}

	return contents, readErr
}

func (r *s3Reader) newS3FileReader(ctx context.Context, file string) (source.ParquetFile, int64, error) {
	bucket := r.getBucketName()

	logger := r.logger.WithContext(ctx)
	logger.Infof("downloading file %s/%s", bucket, file)
	start := time.Now()

	contents, err := r.downloadFile(bucket, file)

	if err != nil {
		return nil, 0, err
	}

	took := time.Now().Sub(start)
	logger.Infof("download took %v for %d bytes; reading file %s/%s", took, len(contents), bucket, file)

	fr, err := parquetBuffer.NewBufferFile(contents)

	return fr, int64(len(contents)), err
}

func (r *s3Reader) ReadFile(ctx context.Context, file string, callback func(chunk ReadResults, lastChunk bool) error) error {
	_, err := r.readFile(ctx, file, callback, nil, 0)

	return err
}

func (r *s3Reader) readFile(
	ctx context.Context,
	file string,
	callback func(chunk ReadResults, lastChunk bool) error,
	rowsSem *semaphore.Weighted,
	rowsSemSize int64,
) (int64, error) {
	fr, fileSize, err := r.newS3FileReader(ctx, file)

	if err != nil {
		return 0, err
	}

	pr, err := reader.NewParquetColumnReader(fr, 4)

	if err != nil {
		return 0, err
	}

	size := int(pr.GetNumRows())

	if rowsSem != nil {
		toAcquire := int64(size)
		if toAcquire > rowsSemSize {
			toAcquire = rowsSemSize
		}

		if err := rowsSem.Acquire(ctx, toAcquire); err != nil {
			return 0, err
		}
		defer rowsSem.Release(toAcquire)
	}

	columnNames := pr.SchemaHandler.ValueColumns
	columns := make(map[string][]interface{})
	rootName := pr.Footer.Schema[0].Name

	for _, colSchema := range columnNames {
		values, _, _, err := pr.ReadColumnByPath(colSchema, size)

		if err != nil {
			return 0, err
		}

		key := strings.ToLower(colSchema[len(rootName)+1:])

		columns[key] = values
	}

	pr.ReadStop()
	if err = fr.Close(); err != nil {
		return 0, err
	}

	for offset := 0; offset < size; offset += r.batchSize {
		remaining := size - offset
		if remaining > r.batchSize {
			remaining = r.batchSize
		}

		results := make(ReadResults, remaining)

		for key, values := range columns {
			for i := 0; i < remaining; i++ {
				if results[i] == nil {
					results[i] = make(ReadResult, len(columns))
				}

				results[i][key] = values[i+offset]
			}
		}

		if err := callback(results, offset+r.batchSize >= size); err != nil {
			return 0, err
		}
	}

	return fileSize, nil
}

func (r *s3Reader) listFilesFromDate(datetime time.Time) ([]string, int64, error) {
	prefix := r.prefixNamingStrategy(r.modelId, datetime)
	files, totalSize, err := r.listFiles(prefix)

	if err != nil {
		return nil, 0, err
	}

	return files, totalSize, err
}

func (r *s3Reader) listFiles(prefix string) ([]string, int64, error) {
	bucket := r.getBucketName()

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	files := make([]string, 0, 128)
	var totalSize int64

	for {
		out, err := r.s3Client.ListObjects(input)

		if err != nil {
			return nil, 0, err
		}

		if len(out.Contents) == 0 {
			break
		}

		for _, obj := range out.Contents {
			files = append(files, *obj.Key)
			totalSize += *obj.Size
		}

		if !*out.IsTruncated {
			break
		}

		input.Marker = out.Contents[len(out.Contents)-1].Key
	}

	return files, totalSize, nil
}

func (r *s3Reader) decode(input interface{}, output interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          "parquet",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			r.decodeTimeMillisHook(), // used to decode firehose parquet time millis (little endian int96)
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)

	if err != nil {
		return errors.Wrap(err, "can not initialize decoder")
	}

	err = decoder.Decode(input)

	if err != nil {
		return errors.Wrap(err, "can not decode input")
	}

	return nil
}

func (r *s3Reader) decodeTimeMillisHook() interface{} {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		return parseInt96Timestamp(data.(string)), nil
	}
}

func (r *s3Reader) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     r.modelId.Project,
		Environment: r.modelId.Environment,
		Family:      r.modelId.Family,
		Application: r.modelId.Application,
	})
}
