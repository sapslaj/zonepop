package aws

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("could not get default AWS config: %w", err)
	}
	return route53.NewFromConfig(cfg), nil
}
