package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
)

type CPUController struct {}

func (cs *CPUController) Name() string {
	return "cpu"
}

// Set sets the resource configuration to the path
func (cs *CPUController) Set(path string, res *ResourceConfig) error {
	// Ensure the path exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %v", err)
	}

	// Set cpu.max
	if res.CpuMax != "" {
		if err := writeToFile(filepath.Join(path, "cpu.max"), res.CpuMax + " 100000"); err != nil {
			return fmt.Errorf("failed to set cpu.max: %v", err)
		}
	}

	// Set cpu.weight
	if res.CpuWeight != "" {
		if err := writeToFile(filepath.Join(path, "cpu.weight"), res.CpuWeight); err != nil {
			return fmt.Errorf("failed to set cpu.weight: %v", err)
		}
	}

	// Set cpu.weight
	if res.CpuWeightNice != "" {
		if err := writeToFile(filepath.Join(path, "cpu.weight.nice"), res.CpuWeightNice); err != nil {
			return fmt.Errorf("failed to set cpu.weight.nice: %v", err)
		}
	}

	// Set cpuset.cpus
	if res.CpuSet != "" {
		if err := writeToFile(filepath.Join(path, "cpuset.cpus"), res.CpuSet); err != nil {
			return fmt.Errorf("failed to set cpuset.cpus: %v", err)
		}
	}

	return nil
}