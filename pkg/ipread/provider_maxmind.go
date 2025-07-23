package ipread

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	urlPkg "net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/oschwald/geoip2-golang"
)

type databaseLoader func(ctx context.Context) (io.ReadCloser, int64, error)

type MaxmindSettings struct {
	Database     string `cfg:"database"`
	S3ClientName string `cfg:"s3_client_name"`
}

type maxmindProvider struct {
	lck      sync.RWMutex
	clk      clock.Clock
	logger   log.Logger
	reader   *geoip2.Reader
	loader   databaseLoader
	name     string
	settings *MaxmindSettings
}

func NewMaxmindProvider(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Provider, error) {
	logger = logger.WithFields(map[string]any{
		"provider_name": name,
	})

	key := fmt.Sprintf("ipread.%s", name)
	settings := &MaxmindSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal maxmind settings for %s: %w", name, err)
	}

	var err error
	var loader databaseLoader

	if loader, err = getDatabaseLoader(ctx, config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not get database loader: %w", err)
	}

	provider := &maxmindProvider{
		clk:      clock.NewRealClock(),
		logger:   logger,
		loader:   loader,
		name:     name,
		settings: settings,
	}

	return provider, nil
}

func (p *maxmindProvider) City(ipAddress net.IP) (*geoip2.City, error) {
	if p.reader == nil {
		return nil, fmt.Errorf("maxmind geo ip reader is not initialized yet. Do you have the refresh mode enabled?")
	}

	p.lck.RLock()
	defer p.lck.RUnlock()

	return p.reader.City(ipAddress)
}

func (p *maxmindProvider) Refresh(ctx context.Context) (err error) {
	var read io.Reader
	var source io.ReadCloser
	var ipreader *geoip2.Reader
	var size int64

	p.logger.Info(ctx, "refreshing maxmind provider %s with database file %s", p.name, p.settings.Database)

	if source, size, err = p.loader(ctx); err != nil {
		return fmt.Errorf("can no read database bytes: %w", err)
	}

	defer func() {
		if closeErr := source.Close(); closeErr != nil {
			err = multierror.Append(err, fmt.Errorf("can not close source reader: %w", closeErr))
		}
	}()

	if strings.HasSuffix(p.settings.Database, ".tar.gz") {
		if read, err = readCompressedDatabase(source); err != nil {
			return fmt.Errorf("can not read compressed database: %w", err)
		}
	} else {
		read = source
	}

	clk := clock.NewRealClock()
	start := clk.Now()

	buf := bytes.NewBuffer(make([]byte, 0, size))
	if _, err = io.Copy(buf, read); err != nil {
		return fmt.Errorf("can not read database bytes: %w", err)
	}

	duration := clk.Since(start)
	p.logger.Info(ctx, "reading of %s with size %d bytes took %s", p.settings.Database, buf.Len(), duration)

	if ipreader, err = geoip2.FromBytes(buf.Bytes()); err != nil {
		return fmt.Errorf("can not open database from memory: %w", err)
	}

	p.lck.Lock()
	defer p.lck.Unlock()
	readerToClose := p.reader
	p.reader = ipreader
	if readerToClose != nil {
		if err := readerToClose.Close(); err != nil {
			return fmt.Errorf("can not close existing reader: %w", err)
		}
	}

	return nil
}

func (p *maxmindProvider) Close() error {
	p.lck.Lock()
	defer p.lck.Unlock()

	if p.reader == nil {
		return nil
	}

	err := p.reader.Close()
	p.reader = nil

	if err != nil {
		return fmt.Errorf("can not close maxmind reader: %w", err)
	}

	return nil
}

func getDatabaseLoader(ctx context.Context, config cfg.Config, logger log.Logger, settings *MaxmindSettings) (databaseLoader, error) {
	var err error
	var url *urlPkg.URL

	if url, err = urlPkg.Parse(settings.Database); err != nil {
		return nil, fmt.Errorf("can not parse database url: %w", err)
	}

	switch url.Scheme {
	case "", "file":
		return readDatabaseLoaderLocal(settings)
	case "s3":
		return readDatabaseLoaderS3(ctx, config, logger, settings)
	default:
		return nil, fmt.Errorf("no database handler found for scheme %s", url.Scheme)
	}
}

func readDatabaseLoaderLocal(settings *MaxmindSettings) (databaseLoader, error) {
	return func(ctx context.Context) (io.ReadCloser, int64, error) {
		var err error
		var file io.ReadCloser
		var info os.FileInfo

		if file, err = os.Open(settings.Database); err != nil {
			return nil, 0, fmt.Errorf("can not open local file %s: %w", settings.Database, err)
		}

		if info, err = os.Stat(settings.Database); err != nil {
			return nil, 0, fmt.Errorf("can not stat local file %s: %w", settings.Database, err)
		}

		return file, info.Size(), err
	}, nil
}

func readDatabaseLoaderS3(ctx context.Context, config cfg.Config, logger log.Logger, settings *MaxmindSettings) (databaseLoader, error) {
	var err error
	var client *s3.Client

	if client, err = gosoS3.ProvideClient(ctx, config, logger, settings.S3ClientName); err != nil {
		return nil, fmt.Errorf("can not get s3 client: %w", err)
	}

	return func(ctx context.Context) (io.ReadCloser, int64, error) {
		var err error
		var url *urlPkg.URL
		var output *s3.GetObjectOutput

		if url, err = urlPkg.Parse(settings.Database); err != nil {
			return nil, 0, fmt.Errorf("can not parse database url: %w", err)
		}

		bucket := url.Host
		key := strings.TrimLeft(url.Path, "/")

		input := &s3.GetObjectInput{
			Bucket: mdl.Box(bucket),
			Key:    mdl.Box(key),
		}

		if output, err = client.GetObject(ctx, input); err != nil {
			return nil, 0, fmt.Errorf("can not get database from bucket %s and key %s: %w", bucket, key, err)
		}

		return output.Body, mdl.EmptyIfNil(output.ContentLength), err
	}, nil
}

func readCompressedDatabase(input io.Reader) (io.Reader, error) {
	var err error
	var uncompressedStream *gzip.Reader
	var tarReader *tar.Reader
	var header *tar.Header

	if uncompressedStream, err = gzip.NewReader(input); err != nil {
		return nil, fmt.Errorf("can not decompress input: %w", err)
	}

	tarReader = tar.NewReader(uncompressedStream)

	for {
		header, err = tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("extractTarGz: Next() failed: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			ext := path.Ext(header.Name)

			if ext != ".mmdb" {
				continue
			}

			return tarReader, nil

		default:
			return nil, fmt.Errorf("unexpected type: %s in %s", string(header.Typeflag), header.Name)
		}
	}

	return nil, fmt.Errorf("no maxmind database file found in the archive")
}
