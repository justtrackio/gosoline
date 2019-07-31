package es

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	"github.com/sha1sum/aws_signing_client"
	"io/ioutil"
	"net/http"
	"time"
)

type ClientV6 struct {
	elasticsearch6.Client
}

var eclV6 = map[string]*ClientV6{}

type clientBuilderV6 func(logger Logger, url string) (*ClientV6, error)

var factoryV6 = map[string]clientBuilderV6{
	"default": func(logger Logger, url string) (*ClientV6, error) {
		cfg := elasticsearch6.Config{
			Addresses: []string{
				url,
			},
		}

		elasticClient, err := elasticsearch6.NewClient(cfg)
		if err != nil {
			logger.Fatal(err, "can't create ES client V6")
			return nil, err
		}

		client := &ClientV6{*elasticClient}
		return client, err
	},

	"aws": func(logger Logger, url string) (*ClientV6, error) {
		client, err := GetAwsClientV6(logger, url)

		return client, err
	},
}

func NewClientV6(config cfg.Config, logger Logger, name string) *ClientV6 {
	clientTypeKey := fmt.Sprintf("es_%s_type", name)
	clientType := config.GetString(clientTypeKey)

	urlKey := fmt.Sprintf("es_%s_endpoint", name)
	url := config.GetString(urlKey)

	client := NewSimpleClientV6(logger, url, clientType)

	templateKey := fmt.Sprintf("es_%s_templates", name)
	templatePath := config.GetStringSlice(templateKey)
	putTemplatesV6(logger, client, name, templatePath)

	return client
}

func NewSimpleClientV6(logger Logger, url string, clientType string) *ClientV6 {
	logger.Info("creating client ", clientType, " for host ", url)

	client, err := factoryV6[clientType](logger, url)

	if err != nil {
		logger.Fatal(err, "error creating the client")
	}

	return client
}

func ProvideClientV6(config cfg.Config, logger Logger, name string) *ClientV6 {
	mtx.Lock()
	defer mtx.Unlock()

	if client, ok := eclV6[name]; ok {
		return client
	}

	eclV6[name] = NewClientV6(config, logger, name)

	return eclV6[name]
}

func GetAwsClientV6(logger Logger, url string) (*ClientV6, error) {
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

	configES := elasticsearch6.Config{
		Addresses: []string{url},
		Transport: awsClient.Transport,
	}

	elasticClient, err := elasticsearch6.NewClient(configES)
	client := &ClientV6{*elasticClient}
	return client, err
}

func putTemplatesV6(logger Logger, client *ClientV6, name string, paths []string) {
	files := getTemplateFiles(logger, paths)

	for _, file := range files {
		buf, err := ioutil.ReadFile(file)

		if err != nil {
			msg := fmt.Sprintf("could not read es-templates file. "+
				"I tried reading %s, but it failed. "+
				"Create the template or set the correct path using es_metric_template", file)
			logger.Fatal(fmt.Errorf(msg), msg)
		}

		// Create template
		res, err := client.Indices.PutTemplate(
			name,
			bytes.NewReader(buf),
		)

		if err != nil {
			msg := fmt.Sprintf("could not put the es-template in file %s for es client %s", file, name)
			logger.Fatal(err, msg)
		}

		defer func() {
			closeError := res.Body.Close()
			if closeError != nil {
				msg := "could not close response reader for PutTemplatesV6"
				logger.Info(msg)
			}
		}()

		if res.IsError() {
			msg := fmt.Sprintf("could not put template from file %s"+
				"Got error from ES: %s, %s", file, res.Status(), res.String())
			logger.Fatal(fmt.Errorf(msg), msg)
		}
	}
}
