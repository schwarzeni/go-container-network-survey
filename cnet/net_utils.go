package cnet

import (
	"fmt"
	"go-container-network-survey/container"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// iptablesAction 操作 iptables的动作
type iptablesAction string

const (
	// ADD 添加
	ADD iptablesAction = "-A"
	// DEL 删除
	DEL iptablesAction = "-D"
)

// createBridgeInterface 创建网络接口
func createBridgeInterface(bridgeName string) (err error) {
	if _, err := net.InterfaceByName(bridgeName); err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	la := netlink.NewLinkAttrs() // 设置 bridge 的属性：名称
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}        // 创建网桥实例
	if err := netlink.LinkAdd(br); err != nil { // 创建网桥
		return fmt.Errorf("Bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return
}

// setInterfaceIP 为网络接口添加网关IP
func setInterfaceIP(bridgeName, rawIP string) (err error) {
	var (
		iface netlink.Link
		ipNet *net.IPNet
	)
	if iface, err = netlink.LinkByName(bridgeName); err != nil { // 根据名称找到相关网桥
		return fmt.Errorf("get interface %s failed, %v", bridgeName, err)
	}
	if ipNet, err = netlink.ParseIPNet(rawIP); err != nil { // 解析用户传入的 string 形式的 ip
		return fmt.Errorf("parse ip %s failed, %v", rawIP, err)
	}
	if err = netlink.AddrAdd(iface, &netlink.Addr{IPNet: ipNet}); err != nil { // 为网桥添加 ip
		return fmt.Errorf("add ip %s to interface %s failed, %v", rawIP, bridgeName, err)
	}
	return
}

// setInterfaceUP 启动网桥
func setInterfaceUP(bridgeName string) (err error) {
	var iface netlink.Link
	if iface, err = netlink.LinkByName(bridgeName); err != nil { // 根据名称找到相关网桥
		return fmt.Errorf("get interface %s failed, %v", bridgeName, err)
	}
	if err = netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("set interface %s up failed, %v", bridgeName, err)
	}
	return
}

// setupIPTables 配置 iptables 相关路由
func setupIPTables(bridgeName string, ipRange *net.IPNet, action iptablesAction) (err error) {
	var iptablesCmds []string
	// 配置 nat 表
	iptablesCmds = append(iptablesCmds,
		// iptables -t nat -A/D POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
		fmt.Sprintf("-t nat %[2]s POSTROUTING -s %[1]s ! -o %[1]s -j MASQUERADE", bridgeName, action))

	// 配置 filter 表
	iptablesCmds = append(iptablesCmds,
		// iptables -t filter -A/D FORWARD -i <bridgeName> -o <bridgeName> -j ACCEPT
		fmt.Sprintf("-t filter %[2]s FORWARD -i %[1]s -o %[1]s -j ACCEPT", bridgeName, action),
		// iptables -t filter -A/D FORWARD -i <bridgeName> ! -o <bridgeName> -j ACCEPT
		fmt.Sprintf("-t filter %[2]s FORWARD -i %[1]s ! -o %[1]s -j ACCEPT", bridgeName, action),
		// iptables -t filter -A/D FORWARD -o <bridgeName> -j ACCEPT
		fmt.Sprintf("-t filter %[2]s FORWARD -o %[1]s -j ACCEPT", bridgeName, action))

	for _, iptablesCmd := range iptablesCmds {
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		if output, err := cmd.Output(); err != nil {
			return fmt.Errorf("create iptables rule %s failed with result %s : %v", iptablesCmd, output, err)
		}
	}
	return
}

// configEndpointIPAddressAndRoute 容器内部配置容器网络、设备 IP 地址和路由信息
func configEndpointIPAddressAndRoute(ep *EndPoint, cinfo *container.Info) (err error) {
	var (
		peerLink netlink.Link
	)
	if peerLink, err = netlink.LinkByName(ep.Device.PeerName); err != nil {
		return fmt.Errorf("get interface %s failed, %v", ep.Device.PeerName, err)
	}

	defer enterContainerNetns(&peerLink, cinfo)()

	// 获取容器的 ip 地址以及网段
	interfaceIP := *ep.Network.IPRange
	interfaceIP.IP = ep.IPAddress

	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("set %s interface ip %s failed, %v", ep.Device.PeerName, interfaceIP.String(), err)
	}

	// 启动容器内部相关的网卡
	for _, ifname := range []string{ep.Device.PeerName, "lo"} {
		if err = setInterfaceUP(ifname); err != nil {
			return fmt.Errorf("set %s interface up failed, %v", ifname, err)
		}
	}

	// 设置容器内部的外部请求都从端点 veth 访问
	// 0.0.0.0/0 表示所有的 IP 地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IPRange.IP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("route add for %s failed %v", ep.Device.PeerName, err)
	}
	return
}

// configPortMapping iptables 配置端口映射信息
func configPortMapping(ep *EndPoint, cinfo *container.Info, action iptablesAction) (err error) {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":") // 默认合法
		// -t nat -A PREROUTING -p tcp --dport 8088 -j DNAT --to 175.18.0.2:80
		// -t filter -A FORWARD -d 175.18.0.2/32 ! -i ns_br -o ns_br -p tcp -m tcp --dport 80 -j ACCEPT
		cmds := []string{
			fmt.Sprintf("-t nat %s PREROUTING -p tcp --dport %s -j DNAT --to %s:%s", action, portMapping[0], ep.IPAddress.String(), portMapping[1]),
			// fmt.Sprintf("-t filter %s FORWARD -d %s ! -i %s -o %s -p tcp -m tcp --dport %s -j ACCEPT", action, ep.Network.IPRange.String(), ep.Network.Name, ep.Network.Name, portMapping[1])}
			fmt.Sprintf("-t filter %s FORWARD -d %s ! -i %s -o %s -p tcp -m tcp --dport %s -j ACCEPT", action, ep.IPAddress.String(), ep.Network.Name, ep.Network.Name, portMapping[1])}

		for _, c := range cmds {
			cmd := exec.Command("iptables", strings.Split(c, " ")...)
			if output, err := cmd.Output(); err != nil {
				log.Printf("iptables output err %v, %v", output, err)
			}
		}
	}
	return
}

// enterContainerNetns 进入容器的 network namespace 中
func enterContainerNetns(enLink *netlink.Link, cinfo *container.Info) func() {
	var (
		f      *os.File
		err    error
		origns netns.NsHandle
	)
	// 获取 net namespace 的文件描述符
	if f, err = os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.PID), os.O_RDONLY, 0); err != nil {
		log.Fatalf("error get container net namespace, %v", err)
	}
	runtime.LockOSThread()

	// 将 veth 移动至容器的 net namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(f.Fd())); err != nil {
		log.Fatalf("error set link netns, %v", err)
	}

	// 获取初始的命名空间
	if origns, err = netns.Get(); err != nil {
		log.Fatalf("error get current netns, %v", err)
	}

	// 进入到容器的 net namespace
	if err = netns.Set(netns.NsHandle(f.Fd())); err != nil {
		log.Fatalf("error set netns, %v", err)
	}

	return func() {
		_ = netns.Set(origns)
		_ = origns.Close()
		runtime.UnlockOSThread()
		_ = f.Close()
	}
}
