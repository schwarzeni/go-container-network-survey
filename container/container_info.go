package container

// Info 容器信息
type Info struct {
	ID          string   // 容器 ID
	PortMapping []string // 容器端口映射
	PID         string   // 容器进程的 ID
}
