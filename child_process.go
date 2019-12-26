package main

import (
	"fmt"
	"os"
	"syscall"
)

func childProcess() {

	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	if err := syscall.Mount("proc", "/proc", "proc", syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, ""); err != nil {
		fmt.Fprintf(os.Stderr, "mount proc error %v", err)
		return
	}

	if err := syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=0755"); err != nil {
		fmt.Fprintf(os.Stderr, "mount tmpfs error %v", err)
		return
	}

	if err := syscall.Exec("/bin/bash", []string{"bash"}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec error %v", err)
		return
	}
}
