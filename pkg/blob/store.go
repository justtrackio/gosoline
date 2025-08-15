package blob

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	PrivateACL    = types.ObjectCannedACLPrivate
	PublicReadACL = types.ObjectCannedACLPublicRead
)

type Object struct {
	ACL  types.ObjectCannedACL
	Body Stream

	Bucket *string

	ContentEncoding *string
	ContentType     *string

	Error error

	Exists bool
	Key    *string
	Prefix *string
	wg     *sync.WaitGroup
}

type CopyObject struct {
	ACL types.ObjectCannedACL

	bucket *string

	ContentEncoding *string
	ContentType     *string

	Error error

	Key          *string
	prefix       *string
	SourceBucket *string
	SourceKey    *string
	wg           *sync.WaitGroup
}

type (
	Batch     []*Object
	CopyBatch []*CopyObject
)

type Settings struct {
	cfg.AppId
	Bucket     string `cfg:"bucket"`
	Region     string `cfg:"region"`
	ClientName string `cfg:"client_name" default:"default"`
	Prefix     string `cfg:"prefix"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Store
type Store interface {
	BucketName() string
	Copy(batch CopyBatch)
	CopyOne(obj *CopyObject) error
	Delete(batch Batch)
	DeletePrefix(ctx context.Context, prefix string) error
	DeleteBucket(ctx context.Context) error
	DeleteOne(obj *Object) error
	ListObjects(ctx context.Context, prefix string) (Batch, error)
	Read(batch Batch)
	ReadOne(obj *Object) error
	Write(batch Batch) error
	WriteOne(obj *Object) error
}

var _ Store = &s3Store{}

type s3Store struct {
	channels *BatchRunnerChannels
	client   gosoS3.Client

	bucket *string
	prefix *string
	region string
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

// NewStore creates a new S3 store with the given configuration and logger.
func NewStore(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Store, error) {
	channels, err := ProvideBatchRunnerChannels(ctx, config, name)
	if err != nil {
		return nil, fmt.Errorf("can not create batch runner channels: %w", err)
	}

	settings, err := ReadStoreSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("can not read store settings for %s: %w", name, err)
	}

	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, settings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client with name %s: %w", settings.ClientName, err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManager(settings, name)); err != nil {
		return nil, fmt.Errorf("can not add life cycle manager: %w", err)
	}

	return NewStoreWithInterfaces(channels, s3Client, settings), nil
}

func NewStoreWithInterfaces(channels *BatchRunnerChannels, client gosoS3.Client, settings *Settings) Store {
	return &s3Store{
		channels: channels,
		client:   client,
		bucket:   mdl.Box(settings.Bucket),
		prefix:   mdl.Box(settings.Prefix),
		region:   settings.Region,
	}
}

func (s *s3Store) BucketName() string {
	return *s.bucket
}

// DeletePrefix deletes all objects in the given prefix.
func (s *s3Store) DeletePrefix(ctx context.Context, prefix string) error {
	batch, err := s.ListObjects(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list blob store objects: %w", err)
	}

	s.Delete(batch)

	return nil
}

// ListObjects lists all keys under a certain prefix of the bucket.
// The prefix of the blob store configuration is already taken into account and must not be part of the prefix parameter
func (s *s3Store) ListObjects(ctx context.Context, prefix string) (Batch, error) {
	var continuationToken *string
	objects := make(Batch, 0)

	// Assemble blob stores prefix and combine it with specified prefix
	pprefix := s.prefix
	if prefix != "" {
		pprefix = mdl.Box(prefix)
		if mdl.EmptyIfNil(s.prefix) != "" {
			pprefix = mdl.Box(fmt.Sprintf("%s/%s", mdl.EmptyIfNil(s.prefix), prefix))
		}
	}

	for {
		loi := s3.ListObjectsV2Input{
			Bucket:            s.bucket,
			Prefix:            pprefix,
			ContinuationToken: continuationToken,
		}

		result, err := s.client.ListObjectsV2(ctx, &loi)
		if err != nil {
			return nil, fmt.Errorf("failed to get object list: %w", err)
		}

		for _, object := range result.Contents {
			// trim excessive blob store prefix from key, excessive '/' is removed in delete runner
			oKey := strings.TrimLeft(strings.TrimPrefix(mdl.EmptyIfNil(object.Key), mdl.EmptyIfNil(s.prefix)), "/")

			objects = append(objects, &Object{
				Bucket: s.bucket,
				Prefix: s.prefix,
				Key:    mdl.Box(oKey),
			})
		}

		if !mdl.EmptyIfNil(result.IsTruncated) || mdl.EmptyIfNil(result.NextContinuationToken) == "" {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return objects, nil
}

func (s *s3Store) ReadOne(obj *Object) error {
	s.Read(Batch{obj})

	return obj.Error
}

func (s *s3Store) Read(batch Batch) {
	wg := &sync.WaitGroup{}
	wg.Add(len(batch))

	for i := 0; i < len(batch); i++ {
		batch[i].Bucket = s.bucket
		batch[i].Prefix = s.prefix
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
		batch[i].Bucket = s.bucket
		batch[i].Prefix = s.prefix
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
		batch[i].Bucket = s.bucket
		batch[i].Prefix = s.prefix
		batch[i].wg = wg
	}

	for i := 0; i < len(batch); i++ {
		s.channels.delete <- batch[i]
	}

	wg.Wait()
}

// DeleteBucket deletes the bucket and all objects in it.
func (s *s3Store) DeleteBucket(ctx context.Context) error {
	err := s.DeletePrefix(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to delete all objects from bucket prior to deleting it: %w", err)
	}

	_, err = s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: s.bucket})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

func (o *Object) GetFullKey() string {
	return getFullKey(o.Prefix, o.Key)
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

func getConfigKey(name string) string {
	return fmt.Sprintf("blob.%s", name)
}

func ReadStoreSettings(config cfg.Config, name string) (*Settings, error) {
	settings := &Settings{}
	key := getConfigKey(name)
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blob store settings for %s: %w", name, err)
	}
	settings.PadFromConfig(config)

	if settings.Bucket == "" {
		settings.Bucket = fmt.Sprintf("%s-%s-%s", settings.Project, settings.Environment, settings.Family)
	}

	if settings.Region == "" {
		s3ClientConfig, err := gosoS3.GetClientConfig(config, settings.ClientName)
		if err != nil {
			return nil, fmt.Errorf("failed to get s3 client config for %s: %w", settings.ClientName, err)
		}

		settings.Region = s3ClientConfig.Settings.Region
	}

	return settings, nil
}
