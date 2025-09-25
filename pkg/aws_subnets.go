package aws_subnets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
)

func GetVPC(awsProfile string, awsRegion string) ([]string, error) {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		return []string{}, err
	}

	// Create EC2 client
	ec2Client := ec2.NewFromConfig(cfg)

	// Prepare input parameters for Describe VPCs API (no filter needed)
	input := &ec2.DescribeVpcsInput{}

	// Call DescribeVpcs API
	resp, err := ec2Client.DescribeVpcs(context.TODO(), input)
	if err != nil {
		return []string{}, err
	}

	// Print information for each VPC
	var vpcIds []string

	for _, vpc := range resp.Vpcs {
		vpcIds = append(vpcIds, *vpc.VpcId)
	}
	return vpcIds, nil
}

func GetSubnetsForVpc(awsProfile string, awsRegion string, vpcID string) ([]types.Subnet, error) {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		return []types.Subnet{}, err
	}

	// Create EC2 client
	ec2Client := ec2.NewFromConfig(cfg)

	// Prepare input parameters for DescribeSubnets API
	input := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	}

	// Call DescribeSubnets API
	resp, err := ec2Client.DescribeSubnets(context.TODO(), input)
	if err != nil {
		panic(fmt.Errorf("failed to describe subnets: %w", err))
	}

	return resp.Subnets, nil
}

func GetAllRegions(awsProfile string) ([]string, error) {
	// Create AWS session using default configuration with us-east-1 to get regions
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return []string{}, err
	}

	// Create EC2 client
	ec2Client := ec2.NewFromConfig(cfg)

	// Call DescribeRegions API
	resp, err := ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return []string{}, err
	}

	var regions []string
	for _, region := range resp.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

type RegionVPCInfo struct {
	Region string
	VPCIds []string
}

func GetVPCsInAllRegions(awsProfile string) ([]RegionVPCInfo, error) {
	regions, err := GetAllRegions(awsProfile)
	if err != nil {
		return []RegionVPCInfo{}, err
	}

	var regionsWithVPCs []RegionVPCInfo
	for _, region := range regions {
		vpcIds, err := GetVPC(awsProfile, region)
		if err != nil {
			// Skip regions where we can't access VPCs (might be disabled)
			continue
		}
		if len(vpcIds) > 0 {
			regionsWithVPCs = append(regionsWithVPCs, RegionVPCInfo{
				Region: region,
				VPCIds: vpcIds,
			})
		}
	}
	return regionsWithVPCs, nil
}
