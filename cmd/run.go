package cmd

import (
	log "github.com/sirupsen/logrus"
	container "minidocker/container"
	cgroups "minidocker/container/cgroups"
	"os"
	"strings"
)

/**
 * @Description: Run command in separate container,
 *	if tty is true, then attach stdin, stdout, stderr to os.Stdin, os.Stdout, os.Stderr
 * @param cmd command to run
 * @param rootDir root directory of the container
 * @param tty attach stdin, stdout, stderr to os.Stdin, os.Stdout, os.Stderr
 * @return void
 */
func Run(cmd []string, rootDir string, volume string, tty bool, res *cgroups.ResourceConfig) {
	parent, writePipe, err := container.NewProcess(cmd, rootDir, volume, tty)
	defer container.DeleteWorkSpace(rootDir, volume)
	if err != nil {
		log.Errorf("Failed to create process: %v", err)
		return
	}

	if err := parent.Start(); err != nil {
		log.Errorf("Failed to start process: %v", err)
	}

	log.Infof("Rescource config: %v", res)
	cgroupManager, err := container.GetCgroupsManager()
	if err == nil {
		cgroupManager.CreateCgroup("minidocker-cgroup")
		cgroupManager.Set("minidocker-cgroup", res)
		cgroupManager.Apply("minidocker-cgroup", parent.Process.Pid)
		defer cgroupManager.Destroy("minidocker-cgroup")
	}

	// send init command to child process
	sendInitCommand(cmd, writePipe)
	_ = parent.Wait()
}

// sendInitCommand 通过writePipe将指令发送给子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("Send init command: %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
