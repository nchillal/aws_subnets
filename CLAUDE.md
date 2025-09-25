# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI application that displays AWS subnet information across all regions in a tabular format. The application allows users to select an AWS profile, then automatically scans all AWS regions to find VPCs and their associated subnets, displaying the results organized by region.

## Architecture

- **main.go**: Entry point that handles user interaction and output formatting
  - Uses `github.com/nchillal/aws_menu` for interactive AWS profile selection
  - Automatically scans all AWS regions for VPCs and subnets
  - Displays results using `tablewriter` with colored output via `fatih/color`
  - Organizes output by region, then by VPC, with subnets sorted alphabetically by name

- **pkg/aws_subnets.go**: Core AWS API functionality
  - `GetVPC()`: Retrieves all VPC IDs for a specified profile/region
  - `GetSubnetsForVpc()`: Gets detailed subnet information for a specific VPC
  - `GetAllRegions()`: Retrieves list of all available AWS regions
  - `GetVPCsInAllRegions()`: Scans all regions and returns only those with VPCs
  - Uses AWS SDK v2 for EC2 operations

## Common Commands

### Build and Run
```bash
go build -o aws_subnets .
./aws_subnets
```

### Run directly
```bash
go run main.go
```

### Get dependencies
```bash
go mod tidy
go mod download
```

### Debug in VS Code
The project includes VS Code launch configuration in `.vscode/launch.json` to debug `main.go`.

## Key Dependencies

- **AWS SDK v2**: Primary AWS API client (`github.com/aws/aws-sdk-go-v2`)
- **AWS SDK v1**: Used for some AWS utilities (`github.com/aws/aws-sdk-go`)
- **Custom AWS Menu**: Interactive profile/region selection (`github.com/nchillal/aws_menu`)
- **Table Writer**: Terminal table formatting (`github.com/olekukonko/tablewriter`)
- **Color**: Terminal color output (`github.com/fatih/color`)

## Development Notes

- The application assumes AWS credentials are configured via AWS profiles
- Error handling uses color-coded output (red for errors, yellow for warnings, cyan for info)
- The main workflow: Profile selection → Multi-region VPC discovery → Subnet listing per region/VPC
- Regions without VPCs are automatically skipped (no output shown)
- Subnet data is sorted by Name tag before display within each VPC
- The application gracefully handles regions that may be disabled or inaccessible
- No tests are currently present in the codebase