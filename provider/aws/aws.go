package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
)

func defaultAWSConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO())
}

func defaultR53Client() (*route53.Client, error) {
	cfg, err := defaultAWSConfig()
	if err != nil {
		return nil, err
	}
	return route53.NewFromConfig(cfg), nil
}
