package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if os.Args[0] == "/proc/self/exe" {
		childProcess()
	}
	var (
		cmd *exec.Cmd
	)
	cmd = exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET}
	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start() failed: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatalf("cmd.Wait %v", err)
	}
}
