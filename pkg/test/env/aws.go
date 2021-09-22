package env

import "github.com/aws/aws-sdk-go-v2/credentials"

const (
	DefaultAccessKeyID     = "gosoline"
	DefaultSecretAccessKey = "gosoline"
	DefaultToken           = ""
)

func GetDefaultStaticCredentials() credentials.StaticCredentialsProvider {
	return credentials.NewStaticCredentialsProvider(DefaultAccessKeyID, DefaultSecretAccessKey, DefaultToken)
}
