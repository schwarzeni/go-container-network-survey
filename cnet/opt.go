package cnet

import (
	"fmt"
	"go-container-network-survey/container"
	"log"
	"net"
	"strings"
)

var (
	drivers                 = map[string]NetworkDriver{}
	networks                = map[string]*Network{}
	defaultNetworksSavePath = "/var/run/mydocker/network/networks.json"
)

func init() {
	// 载入所有网络的信息 networks
	var (
		err      error
		notExist bool
	)
	if notExist, err = loadJSON(&networks, defaultNetworksSavePath); err != nil {
		log.Fatal(err)
	}
	if notExist {
		networks = map[string]*Network{}
	}

	// 载入所有驱动的信息 drivers
	var bridgeDriver = BridgeNetDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
}

// CreateNetwork 创建网络（网桥）
func CreateNetwork(driver, subnet, name string) (err error) {
	// 解析网址
	gw, cidr, _ := net.ParseCIDR(subnet) // 默认输入合法

	// 查看网桥是否存在
	if n, ok := networks[name]; ok == true || (ok == true && n.IPRange.IP.Equal(gw)) {
		return fmt.Errorf("network already exists %s : %s", name, subnet)
	}

	// 查看驱动是否存在
	if _, ok := drivers[driver]; ok == false {
		return fmt.Errorf("driver %s not found", driver)
	}

	if gw, err = ipAllocator.Allocate(cidr); err != nil {
		return fmt.Errorf("allocate ip %s failed: %v", cidr, err)
	}
	cidr.IP = gw

	// 创建网桥
	var newNetwork *Network
	if newNetwork, err = drivers[driver].Create(cidr.String(), name); err != nil {
		return err
	}

	// 记录相关信息
	networks[newNetwork.Name] = newNetwork
	err = dumpJSON(networks, defaultNetworksSavePath)
	return
}

// DeleteNetwork 删除网络（网桥）
// 未考虑的事项：删除网络的时候需要检验是否还存在连接在其上的其他ip
func DeleteNetwork(networkName string) (err error) {
	var (
		nw *Network
		ok bool
	)
	if nw, ok = networks[networkName]; !ok {
		return fmt.Errorf("no such network %s", networkName)
	}
	// WARN: 注意，这里书中有一个bug，不能直接传 nw.IPRange , 这不是网络的范围
	_, ipnet, _ := net.ParseCIDR(nw.IPRange.String())
	if err = ipAllocator.Release(ipnet, &nw.IPRange.IP); err != nil {
		return fmt.Errorf("Error remove network gateway ip %v : %v", &nw.IPRange.IP, err)
	}
	if err = drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error remove network drivererror: %v", err)
	}
	delete(networks, networkName)
	err = dumpJSON(networks, defaultNetworksSavePath)
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

	// 创建网络端点信息
	_, subnet, _ := net.ParseCIDR(network.IPRange.String())
	fmt.Println(subnet)
	// 从网络的IP端中分配容器的IP地址
	if ip, err = ipAllocator.Allocate(subnet); err != nil {
		return fmt.Errorf("failed to allocate ip from range %v : %v", subnet, err)
	}
	// if ip, err = ipAllocator.Allocate(network.IPRange); err != nil {
	// return fmt.Errorf("failed to allocate ip from range %v : %v", *network.IPRange, err)
	// }

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

	// TODO: 修复此处的 magic code，重构记录容器ip地址的代码！！！！！
	cinfo.IP = ip.String() + "/" + strings.Split(subnet.String(), "/")[1] // 记录一下容器的 IP 信息
	return
}

// DisConnect 断开容器到之前创建的网络
func DisConnect(networkName string, cinfo *container.Info) (err error) {
	var (
		network *Network
		ok      bool
	)
	// 从 networks 数组中取到网络的配置信息
	if network, ok = networks[networkName]; !ok {
		return fmt.Errorf("No such network: %s", networkName)
	}
	// 创建网络端点信息
	cip, subnet, _ := net.ParseCIDR(cinfo.IP)
	ep := &EndPoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   cip,
		Network:     network,
		PortMapping: cinfo.PortMapping}

	// 删除 iptables 上的相关信息
	if err = configPortMapping(ep, cinfo, DEL); err != nil {
		return fmt.Errorf("Config port mapping %s failed, %v", cinfo.PortMapping, err)
	}

	fmt.Println("DisConnect", cinfo.IP)
	if err = ipAllocator.Release(subnet, &cip); err != nil {
		return fmt.Errorf("failed to Release ip from range %v : %v", *network.IPRange, err)
	}
	return
}
