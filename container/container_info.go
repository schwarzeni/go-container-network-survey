package container

import "fmt"

// Info 容器信息
type Info struct {
	ID          string   // 容器 ID
	PortMapping []string // 容器端口映射 host_port:container_port
	Port        string   // 服务器监听的端口
	IP          string   // 服务器分配到的IP
	PID         string   // 容器进程的 ID
}

func (containerInfo Info) String() string {
	return fmt.Sprintf("id: %s\npid: %s\ncontainer ip: %s\nport mapping: %s",
		containerInfo.ID, containerInfo.PID, containerInfo.IP, containerInfo.PortMapping)
}
