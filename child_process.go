package main

import (
	"encoding/json"
	"fmt"
	"go-container-network-survey/container"
	"io/ioutil"
	"os"
	"strings"
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

	pipe := os.NewFile(uintptr(3), "pipe") // 默认三个文件句柄，此为之后添加的第四个
	r, _ := ioutil.ReadAll(pipe)
	containerInfo := container.Info{}
	_ = json.Unmarshal(r, &containerInfo)

	if err := syscall.Exec("/usr/bin/python3", strings.Split("python3 -m http.server "+containerInfo.Port, " "), os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec error %v", err)
		return
	}
}
