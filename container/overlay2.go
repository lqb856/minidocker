package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	path "path/filepath"

	log "github.com/sirupsen/logrus"
)

// NewWorkSpace Create an Overlay2 filesystem as container root workspace
func NewWorkSpace(rootPath string, volume string) (string, error) {
	lower, err := createLower(rootPath)
	if err != nil {
		log.Errorf("Failed to create lower dir %s, error: %v", lower, err)
		return "", err
	}
	log.Infof("Lower dir created: %s", lower)

	upper, work, err := createUpperWork(rootPath)
	if err != nil {
		log.Errorf("Failed to create upper and work dir, error: %v", err)
		return "", err
	}
	log.Infof("Upper dir created: %s", upper)
	log.Infof("Work dir created: %s", work)

	mntDir, err := mountOverlayFS(rootPath, lower, upper, work)
	if err != nil {
		log.Errorf("Failed to mount overlay fs, error: %v", err)
		return "", err
	}
	log.Infof("Overlay fs mounted")

	if volume != "" {
		hostPath, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("Failed to extract volume parameter: %v", err)
			return "", err
		}
		err = mountVolume(mntDir, hostPath, containerPath)
		if err != nil {
			log.Errorf("Failed to mount volume: %v", err)
			return "", err
		}
		log.Infof("Volume mounted, host path: %s, container path: %s", hostPath, containerPath)
	}

	return mntDir, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], must split by `:`", volume)
	}

	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}

	return sourcePath, destinationPath, nil
}

// Create lower layer
func createLower(rootURL string) (string, error) {
	busyboxURL := path.Join(rootURL, "busybox")
	busyboxTarURL := path.Join(rootURL, "busybox.tar")

	exist, err := fileExists(busyboxURL)
	if err != nil {
		log.Infof("Failed to judge whether dir %s exists. %v", busyboxURL, err)
		return "", err
	}

	if !exist {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("Failed to make dir %s, error: %v", busyboxURL, err)
			return "", err
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Failed to untar %s, error: %v", busyboxURL, err)
			return "", err
		}
	}
	return busyboxURL, nil
}

func createUpperWork(rootURL string) (string, string, error) {
	upperURL := path.Join(rootURL, "upper")
	if err := os.Mkdir(upperURL, 0777); err != nil {
		log.Errorf("Failed to mkdir dir %s, error: %v", upperURL, err)
		return "", "", err
	}
	workURL := path.Join(rootURL, "work")
	if err := os.Mkdir(workURL, 0777); err != nil {
		log.Errorf("Failed to mkdir dir %s, error: %v", workURL, err)
		return "", "", err
	}
	return upperURL, workURL, nil
}

// mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
func mountOverlayFS(rootURL, lowerURL, upperURL, workURL string) (string, error) {
	mntURL := path.Join(rootURL, "merged")
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("Failed to make merge dir %s, error: %v", mntURL, err)
		return "", err
	}
	log.Infof("Merged dir created: %s", mntURL)

	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work
	dirs := "lowerdir=" + lowerURL + ",upperdir=" + upperURL + ",workdir=" + workURL
	log.Infof("Overlay dirs: %s", dirs)

	// dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Failed to mount overlay fs, error: %v", err)
		return "", err
	}
	return mntURL, nil
}

func mountVolume(mntPath, hostPath, containerPath string) error {
	// create host path if not exist
	if err := os.MkdirAll(hostPath, 0777); err != nil {
		log.Infof("Failed to create host dir %s, error: %v", hostPath, err)
		return err
	}
	// containerPathInHost := /mntPath/containerPath
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.MkdirAll(containerPathInHost, 0777); err != nil {
		log.Infof("Failed to create container dir %s, error: %v", containerPathInHost, err)
		return err
	}
	// mount -o bind /hostPath /containerPath
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Failed to mount volume: %v", err)
		return err
	}
	return nil
}


// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootURL string, volume string) {
	// Must umount volume first!!
	if volume != "" {
		_, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("Failed to extract volume parameter: %v", err)
		}
		mntPath := path.Join(rootURL, "merged")
		umountVolume(mntPath, containerPath)
	}

	umountOverlayFS(rootURL)
	log.Infof("Overlay fs unmounted")
	deleteDirs(rootURL)
	log.Infof("Overlay dirs deleted")
}

func umountVolume(mntPath, containerPath string) {
	containerPathInHost := path.Join(mntPath, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Failed to umount volume path: %v", err)
	}
	log.Infof("Volume path %s umounted", containerPath)
}

func umountOverlayFS(rootURL string) {
	mntURL := path.Join(rootURL, "merged")
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Failed to umount overlay fs, error: %v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("Failed to remove dir %s, error: %v", mntURL, err)
	}
}

func deleteDirs(rootURL string) {
	upperURL := path.Join(rootURL, "upper")
	if err := os.RemoveAll(upperURL); err != nil {
		log.Errorf("Failed to remove upper dir %s, error: %v", upperURL, err)
	}
	workURL := path.Join(rootURL, "work")
	if err := os.RemoveAll(workURL); err != nil {
		log.Errorf("Failed to remove work dir %s, error: %v", workURL, err)
	}
}