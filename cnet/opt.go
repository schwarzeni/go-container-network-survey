package cnet

import (
	"fmt"
	"go-container-network-survey/container"
	"net"
)

var (
	drivers  = map[string]NetworkDriver{}
	networks = map[string]*Network{}
)

func init() {}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) (err error) {
	return
}

// Connect 连接容器到之前创建的网络
func Connect(networkName string, cinfo *container.Info) (err error) {
	var (
		network *Network
		ok      bool
		ip      net.IP
		driver  NetworkDriver
	)

	// 从 networks 数组中取到网络的配置信息
	if network, ok = networks[networkName]; !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}

	// 从网络的IP端中分配容器的IP地址
	if ip, err = ipAllocator.Allocate(network.IPRange); err != nil {
		return fmt.Errorf("failed to allocate ip from range %v : %v", *network.IPRange, err)
	}

	// 创建网络端点信息
	ep := &EndPoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping}

	// 获取驱动
	if driver, ok = drivers[network.Driver]; !ok {
		return fmt.Errorf("No such driver: %s", network.Driver)
	}

	// 配置一端网桥上的 veth 信息
	if err = driver.Connect(network, ep); err != nil {
		return fmt.Errorf("Config veth on bridge failed: %v", err)
	}

	// 到容器内部配置容器网络、设备 IP 地址和路由信息
	if err = configEndpointIPAddressAndRoute(ep, cinfo); err != nil {
		return fmt.Errorf("Config veth in container failed: %v", err)
	}

	// 配置端口映射信息
	if err = configPortMapping(ep, cinfo, ADD); err != nil {
		return fmt.Errorf("Config port mapping %s failed, %v", cinfo.PortMapping, err)
	}
	return
}
