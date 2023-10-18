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

	awsRegion, err := awsMenu.PrintAwsRegionMenu(awsProfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	color.Cyan("\nAWS Profile: %s\n", awsProfile)
	color.Cyan("AWS Region: %s\n", awsRegion)

	vpcId, err := awsSubnets.GetVPC(awsProfile, awsRegion)
	if err != nil {
		color.Red("\n%s", err)
		return
	}

	if len(vpcId) > 0 {
		blue := color.New(color.Bold, color.FgBlue).SprintFunc()
		fmt.Printf(blue("\nVPC ID: %s\n"), vpcId[0])

		subnets, err := awsSubnets.GetSubnetsForVpc(awsProfile, awsRegion, vpcId[0])
		if err != nil {
			fmt.Println(err)
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
		if table.NumLines() > 0 {
			table.Render()
		} else {
			color.Yellow("\nThere is no subnets created for VPC ", vpcId[0])
		}
	} else {
		color.Yellow("\nThere is no VPC created")
	}
}
