package aws_subnets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
)

func GetVPC(awsProfile string, awsRegion string) ([]string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", awsProfile, awsRegion)
	vpcCacheMutex.RLock()
	if vpcIds, exists := vpcCache[cacheKey]; exists {
		if expTime, hasExp := vpcCacheExp[cacheKey]; hasExp && time.Now().Before(expTime) {
			vpcCacheMutex.RUnlock()
			return vpcIds, nil
		}
	}
	vpcCacheMutex.RUnlock()

	// Get cached EC2 client
	ec2Client, err := getEC2Client(awsProfile, awsRegion)
	if err != nil {
		return []string{}, err
	}

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

	// Cache the result
	vpcCacheMutex.Lock()
	vpcCache[cacheKey] = vpcIds
	vpcCacheExp[cacheKey] = time.Now().Add(cacheTTL)
	vpcCacheMutex.Unlock()

	return vpcIds, nil
}

func GetSubnetsForVpc(awsProfile string, awsRegion string, vpcID string) ([]types.Subnet, error) {
	// Get cached EC2 client
	ec2Client, err := getEC2Client(awsProfile, awsRegion)
	if err != nil {
		return []types.Subnet{}, err
	}

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
	// Check cache first
	regionsMutex.RLock()
	if len(regionsCache) > 0 && time.Now().Before(regionsCacheExp) {
		result := make([]string, len(regionsCache))
		copy(result, regionsCache)
		regionsMutex.RUnlock()
		return result, nil
	}
	regionsMutex.RUnlock()

	// Get cached EC2 client for us-east-1
	ec2Client, err := getEC2Client(awsProfile, "us-east-1")
	if err != nil {
		return []string{}, err
	}

	// Call DescribeRegions API
	resp, err := ec2Client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{})
	if err != nil {
		return []string{}, err
	}

	var regions []string
	for _, region := range resp.Regions {
		regions = append(regions, *region.RegionName)
	}

	// Cache the result
	regionsMutex.Lock()
	regionsCache = make([]string, len(regions))
	copy(regionsCache, regions)
	regionsCacheExp = time.Now().Add(regionsCacheTTL)
	regionsMutex.Unlock()

	return regions, nil
}

type RegionVPCInfo struct {
	Region string
	VPCIds []string
}

type VPCSubnetInfo struct {
	VPCId   string
	Subnets []types.Subnet
	Error   error
}

type RegionSubnetInfo struct {
	Region   string
	VPCInfos []VPCSubnetInfo
	Error    error
}

// Client pool for reusing AWS connections
var (
	clientCache = make(map[string]*ec2.Client)
	clientMutex sync.RWMutex

	// Cache for regions list (rarely changes)
	regionsCache    []string
	regionsCacheExp time.Time
	regionsMutex    sync.RWMutex

	// Cache for VPC data (valid for short periods)
	vpcCache     = make(map[string][]string) // key: profile:region
	vpcCacheExp  = make(map[string]time.Time)
	vpcCacheMutex sync.RWMutex

	cacheTTL = 5 * time.Minute // Cache TTL for VPC data
	regionsCacheTTL = 1 * time.Hour // Longer TTL for regions
)

func getEC2Client(awsProfile, region string) (*ec2.Client, error) {
	cacheKey := fmt.Sprintf("%s:%s", awsProfile, region)

	clientMutex.RLock()
	if client, exists := clientCache[cacheKey]; exists {
		clientMutex.RUnlock()
		return client, nil
	}
	clientMutex.RUnlock()

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithSharedConfigProfile(awsProfile),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)

	clientMutex.Lock()
	clientCache[cacheKey] = client
	clientMutex.Unlock()

	return client, nil
}

func GetVPCsInAllRegions(awsProfile string) ([]RegionVPCInfo, error) {
	regions, err := GetAllRegions(awsProfile)
	if err != nil {
		return []RegionVPCInfo{}, err
	}

	type regionResult struct {
		region string
		vpcIds []string
		err    error
	}

	resultChan := make(chan regionResult, len(regions))
	var wg sync.WaitGroup

	// Limit concurrent goroutines to avoid overwhelming AWS API
	semaphore := make(chan struct{}, 10)

	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			vpcIds, err := GetVPC(awsProfile, r)
			resultChan <- regionResult{region: r, vpcIds: vpcIds, err: err}
		}(region)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var regionsWithVPCs []RegionVPCInfo
	for result := range resultChan {
		if result.err != nil {
			// Skip regions where we can't access VPCs (might be disabled)
			continue
		}
		if len(result.vpcIds) > 0 {
			regionsWithVPCs = append(regionsWithVPCs, RegionVPCInfo{
				Region: result.region,
				VPCIds: result.vpcIds,
			})
		}
	}
	return regionsWithVPCs, nil
}

// GetAllSubnetsForRegions fetches all subnets for all VPCs in all regions concurrently
func GetAllSubnetsForRegions(awsProfile string, regionsWithVPCs []RegionVPCInfo) ([]RegionSubnetInfo, error) {
	resultChan := make(chan RegionSubnetInfo, len(regionsWithVPCs))
	var wg sync.WaitGroup

	// Limit concurrent goroutines
	semaphore := make(chan struct{}, 10)

	for _, regionInfo := range regionsWithVPCs {
		wg.Add(1)
		go func(ri RegionVPCInfo) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			vpcInfos := make([]VPCSubnetInfo, len(ri.VPCIds))
			var vpcWg sync.WaitGroup
			vpcSemaphore := make(chan struct{}, 5) // Nested concurrency limit

			for i, vpcId := range ri.VPCIds {
				vpcWg.Add(1)
				go func(idx int, vpc string) {
					defer vpcWg.Done()
					vpcSemaphore <- struct{}{} // Acquire
					defer func() { <-vpcSemaphore }() // Release

					subnets, err := GetSubnetsForVpc(awsProfile, ri.Region, vpc)
					vpcInfos[idx] = VPCSubnetInfo{
						VPCId:   vpc,
						Subnets: subnets,
						Error:   err,
					}
				}(i, vpcId)
			}

			vpcWg.Wait()
			resultChan <- RegionSubnetInfo{
				Region:   ri.Region,
				VPCInfos: vpcInfos,
				Error:    nil,
			}
		}(regionInfo)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []RegionSubnetInfo
	for result := range resultChan {
		results = append(results, result)
	}
	return results, nil
}
