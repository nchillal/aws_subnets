package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	awsMenu "github.com/nchillal/aws_menu"
	awsSubnets "github.com/nchillal/aws_subnets/pkg"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func main() {
	awsProfile, err := awsMenu.PrintAwsProfileMenu()
	if err != nil {
		fmt.Println(err)
		return
	}

	color.Cyan("\nAWS Profile: %s\n", awsProfile)
	color.Cyan("Scanning all regions for VPCs and subnets...\n")

	regionsWithVPCs, err := awsSubnets.GetVPCsInAllRegions(awsProfile)
	if err != nil {
		color.Red("\n%s", err)
		return
	}

	if len(regionsWithVPCs) == 0 {
		color.Yellow("\nNo VPCs found in any region")
		return
	}

	// Process each region with VPCs
	for _, regionInfo := range regionsWithVPCs {
		color.Blue("\n=== Region: %s ===", regionInfo.Region)

		// Process each VPC in the region
		for _, vpcId := range regionInfo.VPCIds {
			blue := color.New(color.Bold, color.FgBlue).SprintFunc()
			fmt.Printf(blue("\nVPC ID: %s\n"), vpcId)

			subnets, err := awsSubnets.GetSubnetsForVpc(awsProfile, regionInfo.Region, vpcId)
			if err != nil {
				color.Red("Error getting subnets for VPC %s: %s", vpcId, err)
				continue
			}

			// Subnet Information
			var tableData [][]string
			for _, subnet := range subnets {
				// Iterate over tags for this subnet
				if len(subnet.Tags) > 0 {
					for _, tag := range subnet.Tags {
						if *tag.Key == "Name" {
							tableData = append(tableData, []string{
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

			if len(tableData) > 0 {
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
				sort.Slice(tableData, func(i, j int) bool { return tableData[i][0] < tableData[j][0] })
				table.AppendBulk(tableData)
				table.Render()
			} else {
				color.Yellow("No subnets found for VPC %s", vpcId)
			}
		}
	}
}
