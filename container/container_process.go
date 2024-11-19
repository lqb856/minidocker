package container

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

/**
 * @Description: Create a new process with separated namespace
 * @param tty attach stdin, stdout, stderr to os.Stdin, os.Stdout, os.Stderr
 * @param command command to run
 * @param rootDir root directory of the container
 * @return *exec.Cmd process, *os.File pipe, error
 */
func NewProcess(command []string, rootDir string, volume string, tty bool) (*exec.Cmd, *os.File, error) {
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
	
	// Use busybox as rootfs.
	mergeDir, err := NewWorkSpace(rootDir, volume)
	if err != nil {
		log.Errorf("Failed to create workspace: %v", err)
		return nil, nil, err
	}
	cmd.Dir = mergeDir

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
	err := setupMount()
	if err != nil {
		log.Errorf("Failed to setup mount: %v", err)
		return err
	}
	
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
	log.Infof("Find executable path: %s", path)

	log.Infof("Executable: %s, Args: %v", path, args)
	// Execute the command.
	// func syscall.Exec(argv0 string, argv []string, envv []string) (err error)
	// argv0: path to the executable
	// argv: arguments to the executable, argv[0] is the executable itself
	if err := syscall.Exec(path, args[:], os.Environ()); err != nil {
		log.Errorf("Failed to exec command %v: %v", args[0], err)
	}

	return nil
}

// ARGS_PIPE is the first user created FD, so it is 3
const ARGS_PIPE_FD = 3
// Read user command from Pipe
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

func setupMount() error {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Failed to get current location: %v", err)
		return err
	}
	log.Infof("Get current location: %s", pwd)

	// Change the root directory to the current location.
	err = pivotRoot(pwd)
	if err != nil {
		log.Errorf("Failed to change root dir: %v", err)
		return err
	}
	log.Infof("Change root directory to %v", pwd)

	// Mount necessary filesystems.
	err = mountNecessary()
	return err
}

func pivotRoot(root string) error {
	// Mount with MS_PRIVATE flag to create a new mount namespace.
	// Mount with MS_REC flag to apply the mount recursively.
	// This will make sure that the mount namespace is isolated from the parent process.
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE | syscall.MS_REC, ""); err != nil {
		log.Fatalf("Failed to make mount private: %v", err)
		return fmt.Errorf("failed to make mount private: %v", err)
	}

	// Rebind root to make new root in different fs.
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND | syscall.MS_REC, ""); err != nil {
		log.Fatalf("Failed to bind mount new root: %v", err)
		return fmt.Errorf("failed to bind mount new root: %v", err)
	}

	// Create rootfs/.pivot_root to store old root
	pivotDir := path.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		log.Errorf("Failed to make pivotDir: %v", err)
		return err
	}

	// pivot_root to new rootfs, now old root is mounted on rootfs/.pivot_root
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		log.Errorf("PivotRoot fail: %v", err)
		return err
	}

	// Change current working directory to rootfs
	if err := syscall.Chdir("/"); err != nil {
		log.Errorf("Change dir to / failed: %v", err)
		return errors.New("change dir to / failed")
	}

	// Unmount the old root
	pivotDir = path.Join("/", ".pivot_root")
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		log.Errorf("Unmount pivot dir failed: %v", err)
		return errors.New("unmount pivot dir failed")
	}
	// delete the pivotDir
	return os.Remove(pivotDir)
}

func mountNecessary() error {
	// Mount /proc, /sys, /dev, /dev/pts, /run
	defaultMountFlags := uintptr(syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV)

	// Mount /proc
	err := syscall.Mount("proc", "/proc", "proc", defaultMountFlags, "")
	if err != nil {
		log.Errorf("Failed to mount /proc: %v", err)
		return fmt.Errorf("failed to mount /proc: %v", err)
	}
	log.Infof("Mounted proc at /proc")

	// Mount /sys
	err = syscall.Mount("sysfs", "/sys", "sysfs", defaultMountFlags, "")
	if err != nil {
		log.Errorf("Failed to mount /sys: %v", err)
		return fmt.Errorf("failed to mount /sys: %v", err)
	}
	log.Infof("Mounted sysfs at /sys")

	// Mount /tmp
	err = syscall.Mount("tmpfs", "/tmp", "tmpfs", syscall.MS_NOSUID | syscall.MS_NODEV, "")
	if err != nil {
		log.Errorf("Failed to mount /tmp: %v", err)
		return fmt.Errorf("failed to mount /tmp: %v", err)
	}
	log.Infof("Mounted tmpfs at /tmp")

	// Mount /dev
	err = syscall.Mount("devtmpfs", "/dev", "devtmpfs", syscall.MS_NOSUID | syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		log.Errorf("Failed to mount /dev: %v", err)
		return fmt.Errorf("failed to mount /dev: %v", err)
	}
	log.Infof("Mounted devtmpfs at /dev")

	// Mount /dev/pts
	err = syscall.Mount("devpts", "/dev/pts", "devpts", syscall.MS_NOSUID | syscall.MS_NOEXEC, "gid=5,mode=620")
	if err != nil {
		log.Errorf("Failed to mount /dev/pts: %v", err)
		return fmt.Errorf("failed to mount /dev/pts: %v", err)
	}
	log.Infof("Mounted devpts at /dev/pts")

	// Mount /dev/shm
	err = syscall.Mount("tmpfs", "/dev/shm", "tmpfs", syscall.MS_NOSUID | syscall.MS_NODEV, "mode=1777")
	if err != nil {
		log.Errorf("Failed to mount /dev/shm: %v", err)
		return fmt.Errorf("failed to mount /dev/shm: %v", err)
	}
	log.Infof("Mounted tmpfs at /dev/shm")
	return nil
}

