package container

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

/**
 * @Description: Create a new process with separated namespace
 * @param tty attach stdin, stdout, stderr to os.Stdin, os.Stdout, os.Stderr
 * @param command command to run
 * @return *exec.Cmd process, *os.File pipe, error
 */
func NewProcess(command []string, tty bool) (*exec.Cmd, *os.File, error) {
	log.Infof("Creating new process, command: %s, tty: %v", command, tty)

	// use Pipe to communicate with the child process.
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		log.Errorf("Failed to create Pipe: %v", err)
		return nil, nil, err
	}

	// Execute "/proc/self/exe" with args "init".
	// In Linux, /proc/self/exe is a symbolic link to the executable file of the current process.
	// So, this command is actually executing the current process with args "init".
	cmd := exec.Command("/proc/self/exe", "init")
	// Set the command's namespace to be different from the parent process.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	// Set the pipe as the extra file descriptor for the command.
	cmd.ExtraFiles = []*os.File{readPipe}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd, writePipe, nil
}

/**
 * @Description: Initialize the container process and run the command
 * @param command command to run
 * @param args command arguments
 * @return error
 */
func InitContainerProcess() error {
	// Mount with MS_PRIVATE flag to create a new mount namespace.
	// Mount with MS_REC flag to apply the mount recursively.
	// This will make sure that the mount namespace is isolated from the parent process.
	syscall.Mount("", "/", "", syscall.MS_PRIVATE | syscall.MS_REC, "")
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	// Mount container's own /proc directory, so that the container can have its own process list.
	_ = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// Read args from the pipe.
	args := readUserCommand()
	if len(args) == 0 {
		return errors.New("no command to run in container") 
	}

	// Find the executable path.
	path, err := exec.LookPath(args[0])
	if err != nil {
		log.Errorf("Cannot find executable: %v", err)
		return err
	}

	// Execute the command.
	if err := syscall.Exec(path, args[1:], os.Environ()); err != nil {
		log.Errorf("Failed to exec command %v: %v", args[0], err)
	}
	return nil
}

const ARGS_PIPE_FD = 3
func readUserCommand() []string {
	pipe := os.NewFile(uintptr(ARGS_PIPE_FD), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		log.Errorf("Failed to read args from Pipe: %v", err)
		return nil
	}
	msgStr := string(msg)
	log.Infof("Read args from Pipe: %s", msgStr)
	return strings.Split(msgStr, " ")
}

