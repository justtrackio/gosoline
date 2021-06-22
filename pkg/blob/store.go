package blob

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/twinj/uuid"
	"strings"
	"sync"
	"time"
)

const (
	PrivateACL    = "private"
	PublicReadACL = "public-read"
)

type Object struct {
	Key    *string
	Body   Stream
	ACL    *string
	Exists bool
	Error  error

	bucket *string
	prefix *string
	wg     *sync.WaitGroup
}

type CopyObject struct {
	Key          *string
	SourceKey    *string
	SourceBucket *string
	ACL          *string
	Error        error

	bucket *string
	prefix *string
	wg     *sync.WaitGroup
}

type Batch []*Object
type CopyBatch []*CopyObject

type Settings struct {
	cfg.AppId
	Bucket string `cfg:"bucket"`
	Prefix string `cfg:"prefix"`
}

//go:generate mockery --name S3API
type S3API interface {
	s3iface.S3API
}

//go:generate mockery --name Store
type Store interface {
	BucketName() string
	Copy(batch CopyBatch)
	CopyOne(obj *CopyObject) error
	CreateBucket()
	Delete(batch Batch)
	DeleteBucket() error
	DeleteOne(obj *Object) error
	Read(batch Batch)
	ReadOne(obj *Object) error
	Write(batch Batch)
	WriteOne(obj *Object) error
}

type s3Store struct {
	logger   log.Logger
	channels *BatchRunnerChannels
	client   s3iface.S3API

	bucket *string
	prefix *string
}

type NamingFactory func() string

var defaultNamingStrategy = func() string {
	y, m, d := time.Now().Date()
	generatedUuid := uuid.NewV4().String()

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

func NewStore(config cfg.Config, logger log.Logger, name string) *s3Store {
	channels := ProvideBatchRunnerChannels(config)
	client := ProvideS3Client(config)

	var settings Settings
	key := fmt.Sprintf("blobstore.%s", name)
	config.UnmarshalKey(key, &settings)
	settings.AppId.PadFromConfig(config)

	if settings.Bucket == "" {
		settings.Bucket = fmt.Sprintf("%s-%s-%s", settings.Project, settings.Environment, settings.Family)
	}

	store := NewStoreWithInterfaces(logger, channels, client, settings)

	autoCreate := config.GetBool("aws_s3_autoCreate")
	if autoCreate {
		store.CreateBucket()
	}

	return store
}

func NewStoreWithInterfaces(logger log.Logger, channels *BatchRunnerChannels, client s3iface.S3API, settings Settings) *s3Store {
	return &s3Store{
		logger:   logger,
		channels: channels,
		client:   client,
		bucket:   mdl.String(settings.Bucket),
		prefix:   mdl.String(settings.Prefix),
	}
}

func (s *s3Store) BucketName() string {
	return *s.bucket
}

func (s *s3Store) CreateBucket() {
	_, err := s.client.CreateBucket(&s3.CreateBucketInput{
		Bucket: s.bucket,
	})

	if isBucketAlreadyExistsError(err) {
		s.logger.Info("s3 bucket %s did already exist", *s.bucket)
	} else if err != nil {
		s.logger.Error("could not create s3 bucket %s: %w", *s.bucket, err)
	} else {
		s.logger.Info("created s3 bucket %s", *s.bucket)
	}
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
	s.Write(Batch{obj})

	return obj.Error
}

func (s *s3Store) Write(batch Batch) {
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

func (s *s3Store) DeleteBucket() error {
	s.logger.Info("purging bucket %s", *s.bucket)

	result, err := s.client.ListObjects(&s3.ListObjectsInput{Bucket: s.bucket})
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

	_, err = s.client.DeleteBucket(&s3.DeleteBucketInput{Bucket: s.bucket})
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

func getFullKey(prefix, key *string) string {
	return fmt.Sprintf("/%s/%s", mdl.EmptyStringIfNil(prefix), mdl.EmptyStringIfNil(key))
}

func (o *CopyObject) getSource() string {
	sourceKey := mdl.EmptyStringIfNil(o.SourceKey)
	if o.SourceBucket == nil {
		sourceKey = getFullKey(o.prefix, o.SourceKey)
		o.SourceBucket = o.bucket
	}
	if !strings.HasPrefix(sourceKey, "/") {
		// we have to avoid having bucket//key as the source as S3 does not find the object like that
		sourceKey = "/" + sourceKey
	}

	return fmt.Sprintf("%s%s", mdl.EmptyStringIfNil(o.SourceBucket), sourceKey)
}

func isBucketAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	if aerr, ok := err.(awserr.Error); ok {
		return aerr.Code() == s3.ErrCodeBucketAlreadyExists ||
			aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou
	}

	return false
}
