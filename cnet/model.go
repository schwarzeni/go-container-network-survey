package cnet

import (
	"net"

	"github.com/vishvananda/netlink"
)

// Network 网络相关配置
type Network struct {
	Name    string     // 网络名
	IPRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

// EndPoint 网络端点
type EndPoint struct {
	ID          string
	Device      netlink.Veth // Veth 设备
	IPAddress   net.IP
	MacAddress  net.HardwareAddr
	PortMapping []string
	Network     *Network
}

// NetworkDriver 网络驱动接口
type NetworkDriver interface {
	Name() string                                         // 驱动名称
	Create(subnet string, name string) (*Network, error)  // 创建网络
	Delete(network Network) error                         // 删除网络
	Connect(network *Network, endpoint *EndPoint) error   // 连接容器端点至网络
	DisConnect(network Network, endpoint *EndPoint) error // 从网络上移除容器网络端点
}
