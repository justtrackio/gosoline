package es

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/olivere/elastic"
	"github.com/sha1sum/aws_signing_client"
	"io/ioutil"
	"net/http"
	"time"
)

var eclV6 = map[string]*elastic.Client{}

type clientBuilderV6 func(logger Logger, url string) (*elastic.Client, error)

var factoryV6 = map[string]clientBuilderV6{
	"default": func(logger Logger, url string) (*elastic.Client, error) {
		client, err := elastic.NewClient(
			elastic.SetURL(url),
			elastic.SetSniff(false),
		)

		return client, err
	},

	"aws": func(logger Logger, url string) (*elastic.Client, error) {
		client, err := GetAwsClientV6(logger, url)

		return client, err
	},
}

func NewClientV6(config cfg.Config, logger Logger, name string) *elastic.Client {
	clientTypeKey := fmt.Sprintf("es_%s_type", name)
	clientType := config.GetString(clientTypeKey)

	urlKey := fmt.Sprintf("es_%s_endpoint", name)
	url := config.GetString(urlKey)

	logger.Info("creating client ", clientType, " for host ", url)
	client, err := factoryV6[clientType](logger, url)

	if err != nil {
		logger.Fatal(err, "error creating the client")
	}

	templateKey := fmt.Sprintf("es_%s_templates", name)
	templatePath := config.GetStringSlice(templateKey)
	putTemplatesV6(logger, client, name, templatePath)

	return client
}

func ProvideClientV6(config cfg.Config, logger Logger, name string) *elastic.Client {
	mtx.Lock()
	defer mtx.Unlock()

	if client, ok := eclV6[name]; ok {
		return client
	}

	eclV6[name] = NewClientV6(config, logger, name)

	return eclV6[name]
}

func GetAwsClientV6(logger Logger, url string) (*elastic.Client, error) {
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

func putTemplatesV6(logger Logger, client *elastic.Client, name string, paths []string) {
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
