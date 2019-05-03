package es

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/olivere/elastic/v7"
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
	Fatal(err error, msg string)
	Info(args ...interface{})
}

var mtx sync.Mutex
var ecl = map[string]*elastic.Client{}

type clientBuilder func(logger Logger, url string) (*elastic.Client, error)

var factory = map[string]clientBuilder{
	"default": func(logger Logger, url string) (*elastic.Client, error) {
		client, err := elastic.NewClient(
			elastic.SetURL(url),
			elastic.SetSniff(false),
		)

		return client, err
	},

	"aws": func(logger Logger, url string) (*elastic.Client, error) {
		client, err := GetAwsClient(logger, url)

		return client, err
	},
}

func NewClient(config cfg.Config, logger Logger, name string) *elastic.Client {
	clientTypeKey := fmt.Sprintf("es_%s_type", name)
	clientType := config.GetString(clientTypeKey)

	urlKey := fmt.Sprintf("es_%s_endpoint", name)
	url := config.GetString(urlKey)

	logger.Info("creating client ", clientType, " for host ", url)
	client, err := factory[clientType](logger, url)

	if err != nil {
		logger.Fatal(err, "error creating the client")
	}

	templateKey := fmt.Sprintf("es_%s_templates", name)
	templatePath := config.GetStringSlice(templateKey)
	putTemplates(logger, client, name, templatePath)

	return client
}

func ProvideClient(config cfg.Config, logger Logger, name string) *elastic.Client {
	mtx.Lock()
	defer mtx.Unlock()

	if client, ok := ecl[name]; ok {
		return client
	}

	ecl[name] = NewClient(config, logger, name)

	return ecl[name]
}

func GetAwsClient(logger Logger, url string) (*elastic.Client, error) {
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
		logger.Fatal(err, "error creating the elastic aws client")
	}

	client, err := elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetScheme("https"),
		elastic.SetHttpClient(awsClient),
		elastic.SetSniff(false),
	)

	return client, err
}

func putTemplates(logger Logger, client *elastic.Client, name string, paths []string) {
	files := getTemplateFiles(logger, paths)

	for _, file := range files {
		bytes, err := ioutil.ReadFile(file)

		if err != nil {
			msg := fmt.Sprintf("could not read es-templates file. "+
				"I tried reading %s, but it failed. "+
				"Create the template or set the correct path using es_metric_template", file)
			logger.Fatal(fmt.Errorf(msg), msg)
		}

		svc := elastic.NewIndicesPutTemplateService(client)
		svc.Name(name)
		svc.BodyString(string(bytes))

		_, err = svc.Do(context.TODO())

		if err != nil {
			msg := fmt.Sprintf("could not put the es-template in file %s for es client %s", file, name)
			logger.Fatal(err, msg)
		}
	}
}

func getTemplateFiles(logger Logger, paths []string) []string {
	files := make([]string, 0)

	for _, p := range paths {
		fileInfo, err := os.Stat(p)

		if err != nil {
			msg := fmt.Sprintf("there was an error with the es-tempates path %s. Does it exist?", p)
			logger.Fatal(err, msg)
		}

		if fileInfo.Mode().IsRegular() {
			files = append(files, p)
			continue
		}

		if !fileInfo.Mode().IsDir() {
			msg := fmt.Sprintf("the es-tempates path %s is neither a file or a directory", p)
			logger.Fatal(err, msg)
		}

		fileInfos, err := ioutil.ReadDir(p)

		if err != nil {
			msg := fmt.Sprintf("could not scan the the es-tempates path %s", p)
			logger.Fatal(err, msg)
		}

		for _, fileInfo := range fileInfos {
			filename := filepath.Join(p, fileInfo.Name())
			scan := getTemplateFiles(logger, []string{filename})

			files = append(files, scan...)
		}
	}

	return files
}
