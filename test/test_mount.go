package main

import (
	"fmt"
	"os"
	"syscall"
	"path/filepath"
	log "github.com/sirupsen/logrus"
)

func main() {
	newRoot := "/home/lqb/go-project/minidocker/overlay/busybox"
	syscall.Chdir("newRoot")

	// 创建必要的目录结构
	dirs := []string{"proc", "sys", "tmp", "dev", "oldroot"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(newRoot, dir), 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// 创建新的 mount namespace
	if err := syscall.Unshare(syscall.CLONE_NEWNS | syscall.CLONE_NEWPID); err != nil {
		log.Fatalf("Failed to unshare namespaces: %v", err)
	}

	// 确保挂载隔离
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE | syscall.MS_REC, ""); err != nil {
		log.Fatalf("Failed to make mount private: %v", err)
	}

	// 绑定新的根文件系统
	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		log.Fatalf("Failed to bind mount new root: %v", err)
	}

	// 切换到新的根文件系统
	oldRoot := filepath.Join(newRoot, "oldroot")
	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		log.Fatalf("Failed to pivot_root: %v", err)
	}

	// 切换到新的根目录
	if err := syscall.Chdir("/"); err != nil {
		log.Fatalf("Failed to change directory to new root: %v", err)
	}

	// 卸载旧根文件系统
	oldRootMount := "/oldroot"
	if err := syscall.Unmount(oldRootMount, syscall.MNT_DETACH); err != nil {
		log.Fatalf("Failed to unmount old root: %v", err)
	}
	if err := os.RemoveAll(oldRootMount); err != nil {
		log.Fatalf("Failed to remove old root: %v", err)
	}

	mountNecessary()

	// 启动隔离环境中的程序
	err := syscall.Exec("/bin/ls", []string{"ls"}, os.Environ())
	if err != nil {
		log.Fatalf("Failed to exec /bin/sh: %v", err)
	}
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
	err = syscall.Mount("devtmpfs", "/dev", "devtmpfs", syscall.MS_NOSUID, "mode=755")
	if err != nil {
		log.Errorf("Failed to mount /dev: %v", err)
		return fmt.Errorf("failed to mount /dev: %v", err)
	}
	log.Infof("Mounted devtmpfs at /dev")

	// Mount /dev/pts
	err = syscall.Mount("devpts", "/dev/pts", "devpts", syscall.MS_NOSUID, "gid=5,mode=620")
	if err != nil {
		log.Errorf("Failed to mount /dev/pts: %v", err)
		return fmt.Errorf("failed to mount /dev/pts: %v", err)
	}
	log.Infof("Mounted devpts at /dev/pts")

	// Mount /dev/shm
	err = syscall.Mount("tmpfs", "/dev/shm", "tmpfs", syscall.MS_NOSUID, "mode=1777")
	if err != nil {
		log.Errorf("Failed to mount /dev/shm: %v", err)
		return fmt.Errorf("failed to mount /dev/shm: %v", err)
	}
	log.Infof("Mounted tmpfs at /dev/shm")
	return nil
}