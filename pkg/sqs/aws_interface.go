package sqs

import "github.com/aws/aws-sdk-go/service/sqs/sqsiface"

//go:generate mockery --name SQSAPI
type SQSAPI interface {
	sqsiface.SQSAPI
}
