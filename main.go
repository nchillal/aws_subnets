package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	aws_subnets "github.com/nchillal/aws_subnets/pkg"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/nchillal/aws_profiles"
	"github.com/olekukonko/tablewriter"
)

func main() {
	// Get list of profiles configured
	profiles, err := aws_profiles.ListAWSProfiles()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	profileSearcher := func(input string, index int) bool {
		profile := profiles[index]
		name := strings.Replace(strings.ToLower(profile), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	// Create a Select template with custom formatting
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "🔥 {{ . | cyan }}",
		Inactive: "  {{ . | cyan }}",
		Selected: "\U0001F336 {{ . | red | cyan }}",
	}

	// Prompt profiles
	promptProfile := promptui.Select{
		Label:        "Select AWS Profile",
		Items:        profiles,
		Size:         len(profiles),
		HideSelected: true,
		Templates:    templates,
		Searcher:     profileSearcher,
	}

	_, awsProfile, err := promptProfile.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("\nAWS Profile: %q\n", awsProfile)

	regions := aws_subnets.ListAWSRegions(awsProfile)

	regionSearcher := func(input string, index int) bool {
		region := regions[index]
		name := strings.Replace(strings.ToLower(region), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}
	// Prompt regions
	promptRegion := promptui.Select{
		Label:        "Select AWS Regions",
		Items:        regions,
		Size:         len(regions),
		HideSelected: true,
		Templates:    templates,
		Searcher:     regionSearcher,
	}

	_, awsRegion, err := promptRegion.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("AWS Region: %q\n", awsRegion)

	vpcId, err := aws_subnets.GetVPC(awsProfile, awsRegion)
	if err != nil {
		fmt.Println("\n", err)
		return
	}

	if len(vpcId) > 0 {
		blue := color.New(color.Bold, color.FgBlue).SprintFunc()
		fmt.Printf(blue("\nVPC ID: %s\n"), vpcId[0])

		subnets, err := aws_subnets.GetSubnetsForVpc(awsProfile, awsRegion, vpcId[0])
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
