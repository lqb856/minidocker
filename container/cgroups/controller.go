package cgroups

import (
	"fmt"
	"os"
)

/**
 * @Description: ResourceConfig is a struct to store resource configuration
 * @param MemoryLimit memory limit, how much memory can be used
 * @param CpuCfsQuota cpu cfs quota, how much cpu time can be used in a time slice
 * @param CpuShare cpu share, how much cpu time can be used in a period
 * @param CpuSet cpu set, which cpu can be used
 */
type ResourceConfig struct {
	MemoryMin     string
	MemoryMax     string
	MemorySwapMax string
	MemoryLow     string
	MemoryHigh    string
	CpuMax        string
	CpuWeight     string
	CpuWeightNice string
	CpuSet        string
}

/**
 * @Description: Subsystem is an interface to set, apply and remove resource configuration
 */
type Controller interface {
	/**
	 * @Description: Name returns the name of the subsystem
	 */
	Name() string

	/**
	 * @Description: Set sets the resource configuration to the path
	 * @param path path to the cgroup
	 * @param res resource configuration
	 * @return error
	 */
	Set(path string, res *ResourceConfig) error
}

// Helper function to write data to a file
func writeToFile(filePath string, value string) error {
	// Open file
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	// Write value
	_, err = file.WriteString(value + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %v", filePath, err)
	}

	return nil
}
