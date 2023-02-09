package blob

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	PrivateACL    = types.ObjectCannedACLPrivate
	PublicReadACL = types.ObjectCannedACLPublicRead
)

type Object struct {
	Key    *string
	Body   Stream
	ACL    types.ObjectCannedACL
	Exists bool
	Error  error

	ContentEncoding *string
	ContentType     *string

	bucket *string
	prefix *string
	wg     *sync.WaitGroup
}

type CopyObject struct {
	Key          *string
	SourceKey    *string
	SourceBucket *string
	ACL          types.ObjectCannedACL
	Error        error

	ContentEncoding *string
	ContentType     *string

	bucket *string
	prefix *string
	wg     *sync.WaitGroup
}

type (
	Batch     []*Object
	CopyBatch []*CopyObject
)

type Settings struct {
	cfg.AppId
	Bucket string `cfg:"bucket"`
	Prefix string `cfg:"prefix"`
}

//go:generate mockery --name Store
type Store interface {
	BucketName() string
	Copy(batch CopyBatch)
	CopyOne(obj *CopyObject) error
	CreateBucket(ctx context.Context) error
	Delete(batch Batch)
	DeleteBucket(ctx context.Context) error
	DeleteOne(obj *Object) error
	Read(batch Batch)
	ReadOne(obj *Object) error
	Write(batch Batch) error
	WriteOne(obj *Object) error
}

var _ Store = &s3Store{}

type s3Store struct {
	logger   log.Logger
	channels *BatchRunnerChannels
	client   gosoS3.Client

	bucket *string
	prefix *string
}

type NamingFactory func() string

var defaultNamingStrategy = func() string {
	y, m, d := time.Now().Date()
	generatedUuid := uuid.New().NewV4()

	return fmt.Sprintf("%d/%02d/%02d/%s", y, m, d, generatedUuid)
}

var namingStrategy = defaultNamingStrategy

func DefaultNamingStrategy() NamingFactory {
	return defaultNamingStrategy
}

func WithNamingStrategy(strategy NamingFactory) {
	namingStrategy = strategy
}

func CreateKey() string {
	return namingStrategy()
}

func NewStore(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*s3Store, error) {
	channels := ProvideBatchRunnerChannels(config)

	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client default: %w", err)
	}

	var settings Settings
	key := fmt.Sprintf("blobstore.%s", name)
	config.UnmarshalKey(key, &settings)
	settings.AppId.PadFromConfig(config)

	if settings.Bucket == "" {
		settings.Bucket = fmt.Sprintf("%s-%s-%s", settings.Project, settings.Environment, settings.Family)
	}

	store := NewStoreWithInterfaces(logger, channels, s3Client, settings)

	autoCreate := dx.ShouldAutoCreate(config)
	if autoCreate {
		if err = store.CreateBucket(ctx); err != nil {
			return nil, fmt.Errorf("can not create bucket: %w", err)
		}
	}

	return store, nil
}

func NewStoreWithInterfaces(logger log.Logger, channels *BatchRunnerChannels, client gosoS3.Client, settings Settings) *s3Store {
	return &s3Store{
		logger:   logger,
		channels: channels,
		client:   client,
		bucket:   mdl.Box(settings.Bucket),
		prefix:   mdl.Box(settings.Prefix),
	}
}

func (s *s3Store) BucketName() string {
	return *s.bucket
}

func (s *s3Store) CreateBucket(ctx context.Context) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: s.bucket,
	})

	if isBucketAlreadyExistsError(err) {
		s.logger.Info("s3 bucket %s did already exist", *s.bucket)
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not create s3 bucket %s: %w", *s.bucket, err)
	}

	s.logger.Info("created s3 bucket %s", *s.bucket)
	return nil
}

func (s *s3Store) ReadOne(obj *Object) error {
	s.Read(Batch{obj})

	return obj.Error
}

func (s *s3Store) Read(batch Batch) {
	wg := &sync.WaitGroup{}
	wg.Add(len(batch))

	for i := 0; i < len(batch); i++ {
		batch[i].bucket = s.bucket
		batch[i].prefix = s.prefix
		batch[i].wg = wg
	}

	for i := 0; i < len(batch); i++ {
		s.channels.read <- batch[i]
	}

	wg.Wait()
}

func (s *s3Store) WriteOne(obj *Object) error {
	if err := s.Write(Batch{obj}); err != nil {
		return obj.Error
	}

	return nil
}

func (s *s3Store) Write(batch Batch) error {
	wg := &sync.WaitGroup{}
	wg.Add(len(batch))

	for i := 0; i < len(batch); i++ {
		batch[i].bucket = s.bucket
		batch[i].prefix = s.prefix
		batch[i].wg = wg
	}

	for i := 0; i < len(batch); i++ {
		s.channels.write <- batch[i]
	}

	wg.Wait()

	var err error
	for i := 0; i < len(batch); i++ {
		if batch[i].Error != nil {
			err = multierror.Append(err, batch[i].Error)
		}
	}

	return err
}

func (s *s3Store) CopyOne(obj *CopyObject) error {
	s.Copy(CopyBatch{obj})

	return obj.Error
}

func (s *s3Store) Copy(batch CopyBatch) {
	wg := &sync.WaitGroup{}
	wg.Add(len(batch))

	for i := 0; i < len(batch); i++ {
		batch[i].bucket = s.bucket
		batch[i].prefix = s.prefix
		batch[i].wg = wg
	}

	for i := 0; i < len(batch); i++ {
		s.channels.copy <- batch[i]
	}

	wg.Wait()
}

func (s *s3Store) DeleteOne(obj *Object) error {
	s.Delete(Batch{obj})

	return obj.Error
}

func (s *s3Store) Delete(batch Batch) {
	wg := &sync.WaitGroup{}
	wg.Add(len(batch))

	for i := 0; i < len(batch); i++ {
		batch[i].bucket = s.bucket
		batch[i].prefix = s.prefix
		batch[i].wg = wg
	}

	for i := 0; i < len(batch); i++ {
		s.channels.delete <- batch[i]
	}

	wg.Wait()
}

func (s *s3Store) DeleteBucket(ctx context.Context) error {
	s.logger.Info("purging bucket %s", *s.bucket)

	result, err := s.client.ListObjects(ctx, &s3.ListObjectsInput{Bucket: s.bucket})
	if err != nil {
		return err
	}

	var batch Batch
	for _, object := range result.Contents {
		batch = append(batch, &Object{
			Key: object.Key,
		})
	}

	s.Delete(batch)

	_, err = s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: s.bucket})
	if err != nil {
		return err
	}

	s.logger.Info("purging bucket %s done", *s.bucket)

	return nil
}

func (o *Object) GetFullKey() string {
	return getFullKey(o.prefix, o.Key)
}

func (o *CopyObject) GetFullKey() string {
	return getFullKey(o.prefix, o.Key)
}

func getFullKey(prefixPtr, keyPtr *string) string {
	key := mdl.EmptyIfNil(keyPtr)
	key = strings.TrimLeft(key, "/")

	prefix := mdl.EmptyIfNil(prefixPtr)
	prefix = strings.TrimRight(prefix, "/")

	fullKey := fmt.Sprintf("%s/%s", prefix, key)
	fullKey = strings.TrimLeft(fullKey, "/")
	return fullKey
}

func (o *CopyObject) getSource() string {
	sourceKey := mdl.EmptyIfNil(o.SourceKey)
	if o.SourceBucket == nil {
		sourceKey = getFullKey(o.prefix, o.SourceKey)
		o.SourceBucket = o.bucket
	}
	if !strings.HasPrefix(sourceKey, "/") {
		// we have to avoid having bucket//key as the source as S3 does not find the object like that
		sourceKey = "/" + sourceKey
	}

	return fmt.Sprintf("%s%s", mdl.EmptyIfNil(o.SourceBucket), sourceKey)
}

func isBucketAlreadyExistsError(err error) bool {
	var bucketAlreadyExists *types.BucketAlreadyExists
	var bucketAlreadyOwnedByYou *types.BucketAlreadyOwnedByYou

	if errors.As(err, &bucketAlreadyExists) || errors.As(err, &bucketAlreadyOwnedByYou) {
		return true
	}

	return false
}
