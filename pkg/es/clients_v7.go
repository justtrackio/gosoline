package es

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/sha1sum/aws_signing_client"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
}

type ClientV7 struct {
	elasticsearch7.Client
}

var mtx sync.Mutex
var ecl = map[string]*ClientV7{}

type clientBuilder func(logger Logger, url string) (*ClientV7, error)

var factory = map[string]clientBuilder{
	"default": func(logger Logger, url string) (*ClientV7, error) {
		cfg := elasticsearch7.Config{
			Addresses: []string{
				url,
			},
		}

		elasticClient, err := elasticsearch7.NewClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("can't create ES client v7: %w", err)
		}

		client := &ClientV7{*elasticClient}
		return client, err
	},

	"aws": func(logger Logger, url string) (*ClientV7, error) {
		client, err := GetAwsClient(logger, url)

		return client, err
	},
}

func NewClient(config cfg.Config, logger Logger, name string) (*ClientV7, error) {
	clientTypeKey := fmt.Sprintf("es_%s_type", name)
	clientType := config.GetString(clientTypeKey)

	urlKey := fmt.Sprintf("es_%s_endpoint", name)
	url := config.GetString(urlKey)

	client, err := NewSimpleClient(logger, url, clientType)
	if err != nil {
		return nil, fmt.Errorf("can not create the client: %w", err)
	}

	templateKey := fmt.Sprintf("es_%s_templates", name)
	templatePath := config.GetStringSlice(templateKey)

	if err = putTemplates(logger, client, name, templatePath); err != nil {
		return nil, fmt.Errorf("can not put templates: %w", err)
	}

	return client, nil
}

func NewSimpleClient(logger Logger, url string, clientType string) (*ClientV7, error) {
	logger.Info("creating client ", clientType, " for host ", url)

	client, err := factory[clientType](logger, url)
	if err != nil {
		return nil, fmt.Errorf("error creating the client: %w", err)
	}

	return client, nil
}

func ProvideClient(config cfg.Config, logger Logger, name string) (*ClientV7, error) {
	mtx.Lock()
	defer mtx.Unlock()

	if client, ok := ecl[name]; ok {
		return client, nil
	}

	var err error
	if ecl[name], err = NewClient(config, logger, name); err != nil {
		return nil, err
	}

	return ecl[name], nil
}

func GetAwsClient(logger Logger, url string) (*ClientV7, error) {
	configTemplate := &aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(endpoints.EuCentral1RegionID),
		LogLevel:                      aws.LogLevel(aws.LogDebug | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			logger.Debug(args...)
		}),
		HTTPClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}

	sess := session.Must(session.NewSession(configTemplate))
	creds := sess.Config.Credentials

	signer := v4.NewSigner(creds)
	awsClient, err := aws_signing_client.New(signer, nil, "es", endpoints.EuCentral1RegionID)

	if err != nil {
		return nil, fmt.Errorf("error creating the elastic aws client: %w", err)
	}

	configES := elasticsearch7.Config{
		Addresses: []string{url},
		Transport: awsClient.Transport,
	}

	elasticClient, err := elasticsearch7.NewClient(configES)
	client := &ClientV7{*elasticClient}

	return client, err
}

func putTemplates(logger Logger, client *ClientV7, name string, paths []string) error {
	files, err := getTemplateFiles(logger, paths)
	if err != nil {
		return fmt.Errorf("could not get template files: %w", err)
	}

	for _, file := range files {
		buf, err := ioutil.ReadFile(file)

		if err != nil {
			return fmt.Errorf("could not read es-templates file %s: %w", file, err)
		}

		// Create template
		res, err := client.Indices.PutTemplate(
			name,
			bytes.NewReader(buf),
		)

		if err != nil {
			return fmt.Errorf("could not put the es-template in file %s for es client %s: %w", file, name, err)
		}

		defer func() {
			closeError := res.Body.Close()
			if closeError != nil {
				msg := "could not close response reader for PutTemplates"
				logger.Info(msg)
			}
		}()

		if res.IsError() {
			return fmt.Errorf("could not put template from file %s: got error from ES: %s, %s", file, res.Status(), res.String())
		}
	}

	return nil
}

func getTemplateFiles(logger Logger, paths []string) ([]string, error) {
	files := make([]string, 0)

	for _, p := range paths {
		fileInfo, err := os.Stat(p)

		if err != nil {
			return nil, fmt.Errorf("there was an error with the es-tempates path %s. Does it exist?: %w", p, err)
		}

		if fileInfo.Mode().IsRegular() {
			files = append(files, p)
			continue
		}

		if !fileInfo.Mode().IsDir() {
			return nil, fmt.Errorf("the es-tempates path %s is neither a file or a directory: %w", p, err)
		}

		fileInfos, err := ioutil.ReadDir(p)

		if err != nil {
			return nil, fmt.Errorf("could not scan the the es-tempates path %s: %w", p, err)
		}

		for _, fileInfo := range fileInfos {
			filename := filepath.Join(p, fileInfo.Name())

			scan, err := getTemplateFiles(logger, []string{filename})
			if err != nil {
				return nil, fmt.Errorf("could not get template files: %w", err)
			}

			files = append(files, scan...)
		}
	}

	return files, nil
}
