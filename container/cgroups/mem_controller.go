package cgroups

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type MemoryController struct {}

func (ms *MemoryController) Name() string {
	return "memory"
}

// Set sets the resource configuration to the path
func (ms *MemoryController) Set(path string, res *ResourceConfig) error {
	// Ensure the path exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %v", err)
	}

	// Set memory.max
	if res.MemoryMax != "" {
		if err := writeToFile(filepath.Join(path, "memory.max"), res.MemoryMax); err != nil {
			return fmt.Errorf("failed to set memory.max: %v", err)
		}
		log.Infof("Set memory.max: %s", res.MemoryMax)
	}

	// Set memory.min
	if res.MemoryMin != "" {
		if err := writeToFile(filepath.Join(path, "memory.min"), res.MemoryMin); err != nil {
			return fmt.Errorf("failed to set memory.min: %v", err)
		}
		log.Infof("Set memory.min: %s", res.MemoryMin)
	}

	// Set memory.swap.max
	if res.MemorySwapMax != "" {
		if err := writeToFile(filepath.Join(path, "memory.swap.max"), res.MemorySwapMax); err != nil {
			return fmt.Errorf("failed to set memory.swap.max: %v", err)
		}
		log.Infof("Set memory.swap.max: %s", res.MemorySwapMax)
	}

	// Set memory.low
	if res.MemoryLow != "" {
		if err := writeToFile(filepath.Join(path, "memory.low"), res.MemoryLow); err != nil {
			return fmt.Errorf("failed to set memory.low: %v", err)
		}
		log.Infof("Set memory.low: %s", res.MemoryLow)
	}

	// Set memory.high
	if res.MemoryHigh != "" {
		if err := writeToFile(filepath.Join(path, "memory.high"), res.MemoryHigh); err != nil {
			return fmt.Errorf("failed to set memory.high: %v", err)
		}
		log.Infof("Set memory.high: %s", res.MemoryHigh)
	}

	// TODO(lqb): Set other memory constraints

	return nil
}
