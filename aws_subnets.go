package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func getVPC() []string {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("failed to load AWS configuration")
	}

	// Create EC2 client
	ec2_client := ec2.NewFromConfig(cfg)

	// Prepare input parameters for DescribeVpcs API (no filter needed)
	input := &ec2.DescribeVpcsInput{}

	// Call DescribeVpcs API
	resp, err := ec2_client.DescribeVpcs(context.TODO(), input)
	if err != nil {
		panic(fmt.Errorf("failed to describe VPCs: %w", err))
	}

	// Print information for each VPC
	var vpc_ids []string

	for _, vpc := range resp.Vpcs {
		vpc_ids = append(vpc_ids, *vpc.VpcId)
	}
	return vpc_ids
}

func getSubnetsForVpc(vpcID string) ([]types.Subnet, error) {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("failed to load AWS configuration")
	}

	// Create EC2 client
	ec2_client := ec2.NewFromConfig(cfg)

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
	resp, err := ec2_client.DescribeSubnets(context.TODO(), input)
	if err != nil {
		panic(fmt.Errorf("failed to describe subnets: %w", err))
	}

	return resp.Subnets, nil
}

func main() {
	vpc_id := getVPC()
	fmt.Printf("\nVPC ID: %s\n\n", vpc_id[0])
	subnets, err := getSubnetsForVpc(vpc_id[0])
	if err != nil {
		fmt.Println(err)
	}
	// Subnet Information
	table_data := [][]string{}
	for _, subnet := range subnets {
		// Iterate over tags for this subnet
		if len(subnet.Tags) > 0 {
			for _, tag := range subnet.Tags {
				if *tag.Key == "Name" {
					table_data = append(table_data, []string{
						*tag.Value,
						*subnet.SubnetId,
						*subnet.CidrBlock,
						*subnet.AvailabilityZone,
					})
				}
			}
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Name",
		"Subnet ID",
		"CIDR Block",
		"Availability Zone",
	})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)
	sort.Slice(table_data, func(i, j int) bool { return table_data[i][0] < table_data[j][0] })
	table.AppendBulk(table_data)
	if table.NumLines() > 0 {
		table.Render()
	} else {
		color.Yellow("\nThere is no EC2 instance created")
	}
}
