package main

import (
	"encoding/json"
	"fmt"
	"go-container-network-survey/cnet"
	"go-container-network-survey/container"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	if os.Args[0] == "/proc/self/exe" {
		childProcess()
		return
	}
	// -cnet mynet bridge 175.18.0.0/16
	if os.Args[1] == "-cnet" {
		name := os.Args[2]
		driver := os.Args[3]
		subnet := os.Args[4]
		if err := cnet.CreateNetwork(driver, subnet, name); err != nil {
			log.Fatal(err)
		}
		return
	}
	// -dnet mynet
	if os.Args[1] == "-dnet" {
		if err := cnet.DeleteNetwork(os.Args[2]); err != nil {
			log.Fatal(err)
		}
		return
	}
	var (
		cmd           *exec.Cmd
		containerInfo = container.Info{}
	)
	cmd = exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET}

	// 用于父子进程之间同步消息
	r, w, _ := os.Pipe()
	cmd.ExtraFiles = append(cmd.ExtraFiles, r)

	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start() failed: %v", err)
	}

	// 传递一些参数到克隆出的子进程中
	containerInfo.PID = strconv.Itoa(cmd.Process.Pid)
	containerInfo.ID = "2233"
	containerInfo.Port = "8089"
	byteInfo, _ := json.Marshal(&containerInfo)
	_, _ = w.Write(byteInfo)
	_ = w.Close()

	// 处理退出的情况
	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-ch:
		fmt.Printf("[ID: %s]shutdown process %s ...", containerInfo.ID, containerInfo.PID)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatalf("cmd.Wait %v", err)
	}
}
