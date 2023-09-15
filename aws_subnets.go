package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/nchillal/aws_profiles"
	"github.com/olekukonko/tablewriter"
)

func getVPC(awsProfile string, awsRegion string) []string {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(awsRegion),
	)
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

func getSubnetsForVpc(awsProfile string, awsRegion string, vpcID string) ([]types.Subnet, error) {
	// Create AWS session using default configuration
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(awsRegion),
	)
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
	// Get list of profiles configured
	profiles, err := aws_profiles.ListAWSProfiles()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// Create a Select template with custom formatting
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F336 {{ . | cyan }}",
		Inactive: "  {{ . | cyan }}",
		Selected: "\U0001F336 {{ . | red | cyan }}",
	}

	// Prompt profiles
	prompt_profile := promptui.Select{
		Label:        "Select AWS Profile",
		Items:        profiles,
		Size:         len(profiles),
		HideSelected: true,
		Templates:    templates,
	}

	_, awsProfile, err := prompt_profile.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("\nAWS Profile: %q\n", awsProfile)

	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
	)
	if err != nil {
		fmt.Println("Error loading AWS SDK configuration:", err)
		return
	}

	// Create an EC2 client
	ec2_client := ec2.NewFromConfig(cfg)

	// Call DescribeRegions to get a list of regions
	resp, err := ec2_client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{})
	if err != nil {
		fmt.Println("Error describing regions:", err)
		return
	}

	// Get list of regions
	regions := make([]string, 0)
	for _, region := range resp.Regions {
		regions = append(regions, *region.RegionName)
	}
	// Prompt regions
	prompt_region := promptui.Select{
		Label:        "Select AWS Regions",
		Items:        regions,
		Size:         len(regions),
		HideSelected: true,
		Templates:    templates,
	}

	_, awsRegion, err := prompt_region.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("AWS Region: %q\n", awsRegion)

	vpc_id := getVPC(awsProfile, awsRegion)
	if len(vpc_id) > 0 {
		blue := color.New(color.Bold, color.FgBlue).SprintFunc()
		fmt.Println("\n", blue("VPC ID:"), vpc_id[0], "\n")

		subnets, err := getSubnetsForVpc(awsProfile, awsRegion, vpc_id[0])
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
							strconv.FormatInt(int64(*subnet.AvailableIpAddressCount), 10),
							strconv.FormatBool(*subnet.DefaultForAz),
						})
					}
				}
			}
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{
			"Name",
			"Subnet ID",
			"CIDR Block",
			"Availability Zone",
			"Available Ip Count",
			"Default For Az",
		})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false)
		sort.Slice(table_data, func(i, j int) bool { return table_data[i][0] < table_data[j][0] })
		table.AppendBulk(table_data)
		if table.NumLines() > 0 {
			table.Render()
		} else {
			color.Yellow("\nThere is no subnets created for VPC ", vpc_id[0])
		}
	} else {
		color.Yellow("\nThere is no VPC created")
	}
}
