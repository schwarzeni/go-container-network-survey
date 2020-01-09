package cnet

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// BridgeNetDriver bridge网络驱动实现
type BridgeNetDriver struct{}

// Name 驱动名称
func (driver *BridgeNetDriver) Name() (name string) {
	return "bridge"
}

// Create 创建网络
func (driver *BridgeNetDriver) Create(subnet string, name string) (n *Network, err error) {
	var (
		ip      net.IP
		ipRange *net.IPNet
	)
	// 解析用户输入
	if ip, ipRange, err = net.ParseCIDR(subnet); err != nil {
		return nil, fmt.Errorf("parse %s failed, %v", subnet, err)
	}
	ipRange.IP = ip

	n = &Network{ // 初始化网络对象
		Name:    name,
		IPRange: ipRange,
		Driver:  driver.Name(),
	}

	if err = driver.initBridge(n); err != nil {
		return nil, fmt.Errorf("init bridge %s failed with %v", name, *n)
	}
	return
}

// Delete 删除网络
func (driver *BridgeNetDriver) Delete(network Network) (err error) {
	var (
		br         netlink.Link
		bridgeName = network.Name
	)
	if br, err = netlink.LinkByName(bridgeName); err != nil { // 找到相关的网络接口
		return fmt.Errorf("get interface %s failed, %v", bridgeName, err)
	}
	if err = netlink.LinkDel(br); err != nil { // 删除网络接口
		return fmt.Errorf("delete interface %s failed, %v", bridgeName, err)
	}
	if err = setupIPTables(bridgeName, network.IPRange, DEL); err != nil { // 删除 iptables 上相关路由规则
		return fmt.Errorf("setupIPTables( %s, %v ), %v", bridgeName, network.IPRange, err)
	}
	return
}

// Connect 连接容器端点至网桥
func (driver *BridgeNetDriver) Connect(network *Network, endpoint *EndPoint) (err error) {
	var (
		bridgeName = network.Name
		br         netlink.Link
	)
	if br, err = netlink.LinkByName(bridgeName); err != nil {
		return fmt.Errorf("get interface %s failed, %v", bridgeName, err)
	}
	la := netlink.NewLinkAttrs()      // 创建 veth 接口的配置
	la.Name = endpoint.ID[:5]         // Linux 接口名限制，取前 5 位
	la.MasterIndex = br.Attrs().Index // 设置 master 属性，将 veth 一端挂在网络对应的 Linux bridge 上
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5]}
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error add endpoint device: %v", err)
	}
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error set endpoint device up: %v", err)
	}
	return
}

// DisConnect 从网络上移除容器网络端点
func (driver *BridgeNetDriver) DisConnect(network Network, endpoint *EndPoint) (err error) {
	return
}

func (driver *BridgeNetDriver) initBridge(n *Network) (err error) {
	var (
		bridgeName = n.Name
		gatewayIP  = *n.IPRange
	)
	gatewayIP.IP = n.IPRange.IP

	// 1. 创建Bridge虚拟设备
	if err = createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("createBridgeInterface( %s ) , %v", bridgeName, err)
	}

	// 2. 设置Bridge设备的地址和路由
	if err = setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("setInterfaceIP( %s, %s ), %v", bridgeName, gatewayIP.String(), err)
	}

	// 3. 启动Bridge设备
	if err = setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("setInterfaceUP( %s ), %v", bridgeName, err)
	}

	// 4. 设置iptables的SNAT规则
	if err = setupIPTables(bridgeName, n.IPRange, ADD); err != nil {
		return fmt.Errorf("setupIPTables( %s, %v ), %v", bridgeName, n.IPRange, err)
	}
	return
}
