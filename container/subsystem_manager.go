package container

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	cgroups "minidocker/container/cgroups"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

type CgroupsManager struct {
	mutex 			sync.Mutex
	cgroupsRoot string
	cgroups     map[string]struct{}
	controllers []cgroups.Controller
}

// singleton
var (
	once            sync.Once
	globalCgroupMgr *CgroupsManager
)

func GetCgroupsManager() (*CgroupsManager, error) {
	var initErr error
	// init globalCgroupMgr
	once.Do(func() {
		cgroupsRoot, err := findCgroups2Mountpoint()
		if err != nil {
			log.Errorf("failed to find cgroup2 mountpoint err: %v\n", err)
			initErr = errors.New("failed to find cgroup2 mountpoint")
			return
		}

		globalCgroupMgr = &CgroupsManager{
			cgroupsRoot: cgroupsRoot,
			cgroups: make(map[string]struct{}),
			controllers: []cgroups.Controller{
				&cgroups.MemoryController{},
				&cgroups.CPUController{},
			},
		}
	})

	if initErr != nil {
		log.Printf("failed to init cgroup manager err: %v\n", initErr)
		return nil, initErr
	}

	log.Info("cgroup manager initialized")
	return globalCgroupMgr, nil
}

func (m *CgroupsManager) CreateCgroup(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.cgroups[name]; ok {
		log.Infof("cgroup %s already exists\n", name)
		return nil
	}

	fullPath := path.Join(m.cgroupsRoot, name)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		log.Errorf("failed to create cgroup err: %v\n", err)
		return err
	}
	log.Info("cgroup created:", fullPath)
	m.cgroups[name] = struct{}{}
	return nil
}

func (m *CgroupsManager) Apply(cg_name string, pid int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.cgroups[cg_name]; !ok {
		log.Errorf("failed to apply cgroup, cgroup %s not found\n", cg_name)
		return errors.New("cgroup not found")
	}

	full_path := path.Join(m.cgroupsRoot, cg_name)
	procsFile := path.Join(full_path, "/cgroup.procs")
	return os.WriteFile(procsFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func (m *CgroupsManager) Destroy(cg_name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, ok := m.cgroups[cg_name]; !ok {
		log.Infof("cgroup %s not found\n", cg_name)
		return nil
	}
	full_path := path.Join(m.cgroupsRoot, cg_name)
	if err := os.RemoveAll(full_path); err != nil {
		delete(m.cgroups, cg_name)
		log.Info("cgroup destroyed:", full_path)
		return nil
	} else {
		return err
	}
}

func (m *CgroupsManager) Set(cg_name string, res *cgroups.ResourceConfig) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.cgroups[cg_name]; !ok {
		log.Errorf("failed to set cgroup, cgroup %s not found\n", cg_name)
		return errors.New("cgroup not found")
	}

	full_path := path.Join(m.cgroupsRoot, cg_name)
	for _, controller := range m.controllers {
		if err := controller.Set(full_path, res); err != nil {
			log.Errorf("failed to set cgroup %s err: %v\n", cg_name, err)
			return err
		}
	}
	return nil
}

// Index of the mountpoint in the fields of /proc/self/mountinfo
const mountPointIndex = 4

/**
 * @Description: findSubsystemMountpoint finds the mountpoint of the subsystem
 * @Note: This function reads is deprecated because we use cgroup2
 * @param subsystem subsystem name
 * @return string, error
 */
func findSubsystemMountpoint(subsystem string) (string, error) {
	// /proc/self/mountinfo is a file that contains information about mount points in the system
	// "cat /proc/self/mountinfo" will show the content of this file
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		log.Error("failed to open /proc/self/mountinfo err:", err)
		return "", err
	}
	defer f.Close()

	// find target subsystem path
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		subsystems := strings.Split(fields[len(fields)-1], ",")
		for _, opt := range subsystems {
			if opt == subsystem {
				return fields[mountPointIndex], nil
			}
		}
	}

	if err = scanner.Err(); err != nil {
		log.Error("read err:", err)
		return "", err
	}
	return "", nil
}

/**
 * @Description: findCgroupsMountpoint finds the mountpoint of cgroups2
 * @return string, error
 */
func findCgroups2Mountpoint() (string, error) {
	// /proc/self/mountinfo is a file that contains information about mount points in the system
	// "cat /proc/self/mountinfo" will show the content of this file
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		log.Error("failed to open /proc/self/mountinfo err:", err)
		return "", err
	}
	defer f.Close()

	// find cgroup2 path
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, field := range fields {
			if field == "cgroup2" {
				log.Info("cgroup2 mountpoint:", fields[mountPointIndex])
				return fields[mountPointIndex], nil
			}
		}
	}

	if err = scanner.Err(); err != nil {
		log.Error("read err:", err)
		return "", err
	}
	return "", errors.New("cgroup2 mountpoint not found")
}
