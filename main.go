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
	"strings"
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
	// -run <id> <network_name> <host:dst>
	var (
		cmd           *exec.Cmd
		containerInfo = container.Info{}
		containerID   = os.Args[2]
		networkName   = os.Args[3]
		PortMapping   = os.Args[4]
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
	containerInfo.ID = containerID
	containerInfo.Port = strings.Split(PortMapping, ":")[1]
	containerInfo.PortMapping = []string{PortMapping}

	if err := cnet.Connect(networkName, &containerInfo); err != nil {
		log.Fatal(err)
	}

	byteInfo, _ := json.Marshal(&containerInfo)
	_, _ = w.Write(byteInfo)
	// 从这里开始启动容器内的 web 服务
	_ = w.Close()

	log.Printf("start container[%s]:\n%v\n\nrun this command to enter container:\nnsenter --target %s --mount --uts --ipc --net --pid\n\n",
		containerInfo.ID, containerInfo, containerInfo.PID)

	// 处理退出的情况
	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-ch:
		fmt.Printf("\n[ID: %s]shutdown process %s, clear settings ...", containerInfo.ID, containerInfo.PID)
		cnet.DisConnect(networkName, &containerInfo)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatalf("cmd.Wait %v", err)
	}
}
