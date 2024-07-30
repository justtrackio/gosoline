package ipread

import (
	"archive/tar"
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

type databaseLoader func(ctx context.Context) (io.ReadCloser, error)

type maxmindProvider struct {
	lck         sync.RWMutex
	clk         clock.Clock
	logger      log.Logger
	reader      *geoip2.Reader
	loader      func(ctx context.Context) (io.ReadCloser, error)
	currentFile string
	name        string
	settings    *MaxmindSettings
}

func NewMaxmindProvider(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Provider, error) {
	logger = logger.WithFields(map[string]interface{}{
		"provider_name": name,
	})

	key := fmt.Sprintf("ipread.%s", name)
	settings := &MaxmindSettings{}
	config.UnmarshalKey(key, settings)

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
		return nil, fmt.Errorf("maxmind geo ip reader is not initialized yet. do you have the refresh mode enabled?")
	}

	p.lck.RLock()
	defer p.lck.RUnlock()

	return p.reader.City(ipAddress)
}

func (p *maxmindProvider) Refresh(ctx context.Context) (err error) {
	var file *os.File
	var read io.Reader
	var source io.ReadCloser
	var written int64
	var ipreader *geoip2.Reader

	oldFile := p.currentFile
	p.logger.Info("refreshing maxmind provider %s with database file %s", p.name, p.settings.Database)

	if source, err = p.loader(ctx); err != nil {
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

	if file, err = os.CreateTemp("", "ipread_database"); err != nil {
		return fmt.Errorf("can not create temporary file: %w", err)
	}
	p.currentFile = file.Name()

	clk := clock.NewRealClock()
	start := clk.Now()

	if written, err = io.Copy(file, read); err != nil {
		return fmt.Errorf("can not write database file: %w", err)
	}

	duration := clk.Since(start)
	p.logger.Info("reading of %s with size %d bytes took %s", p.settings.Database, written, duration)

	if ipreader, err = geoip2.Open(p.currentFile); err != nil {
		return fmt.Errorf("can not open database file %s: %w", p.currentFile, err)
	}

	p.lck.Lock()
	p.reader = ipreader
	p.lck.Unlock()

	return p.removeDatabaseFile(oldFile)
}

func (p *maxmindProvider) Close() error {
	return p.removeDatabaseFile(p.currentFile)
}

func (p *maxmindProvider) removeDatabaseFile(file string) error {
	if file == "" {
		return nil
	}

	if err := os.Remove(file); err != nil {
		return fmt.Errorf("can not remove old database file %s: %w", file, err)
	}

	p.logger.Info("removed old database file %s", file)

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
	return func(ctx context.Context) (io.ReadCloser, error) {
		var err error
		var file io.ReadCloser

		if file, err = os.Open(settings.Database); err != nil {
			return nil, fmt.Errorf("can not open local file %s: %w", settings.Database, err)
		}

		return file, err
	}, nil
}

func readDatabaseLoaderS3(ctx context.Context, config cfg.Config, logger log.Logger, settings *MaxmindSettings) (databaseLoader, error) {
	var err error
	var client *s3.Client

	if client, err = gosoS3.ProvideClient(ctx, config, logger, settings.S3ClientName); err != nil {
		return nil, fmt.Errorf("can not get s3 client: %w", err)
	}

	return func(ctx context.Context) (io.ReadCloser, error) {
		var err error
		var url *urlPkg.URL
		var output *s3.GetObjectOutput

		if url, err = urlPkg.Parse(settings.Database); err != nil {
			return nil, fmt.Errorf("can not parse database url: %w", err)
		}

		bucket := url.Host
		key := strings.TrimLeft(url.Path, "/")

		input := &s3.GetObjectInput{
			Bucket: mdl.Box(bucket),
			Key:    mdl.Box(key),
		}

		if output, err = client.GetObject(ctx, input); err != nil {
			return nil, fmt.Errorf("can not get database from bucket %s and key %s: %w", bucket, key, err)
		}

		return output.Body, err
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
