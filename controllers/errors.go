package controllers

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

func IsAWSNotFound(err error) bool {
	if err == nil {
		return false
	}
	if aerr, ok := err.(awserr.Error); ok {
		return strings.Index(aerr.Error(), "does not exist") > 0
	}

	return false
}
