package cnet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.json"

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// IPAM 存放IP分配信息
type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

// Allocate 获取一个IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	if err = ipam.load(); err != nil {
		return nil, fmt.Errorf("load network info from %s failed, %v", ipam.SubnetAllocatorPath, err)
	}

	one, bits := subnet.Mask.Size() // for 127.0.0.0/8 --> one=8, bits=32

	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(bits-one))
	}

	netSet := (*ipam.Subnets)[subnet.String()]
	subnetStr := subnet.String()
	for c := range netSet {
		if c >= len(netSet) {
			break
		}
		if netSet[c] == '0' {
			ipalloc := []byte(netSet)
			ipalloc[c] = '1'
			netSet = string(ipalloc)
			ip = make([]byte, len(subnet.IP))
			copy(ip, subnet.IP)
			for t := uint(4); t > 0; t-- {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			if ip[3] != 255 {
				ip[3]++
				break
			}
			ip = net.IP{}
		}
	}
	(*ipam.Subnets)[subnetStr] = netSet

	if len(ip) != 4 || ip[3] == 0 {
		return nil, fmt.Errorf("network %s has been used up", subnet.String())
	}

	if err = ipam.dump(); err != nil {
		return nil, fmt.Errorf("dump network info from %s failed, %v", ipam.SubnetAllocatorPath, err)
	}
	return
}

// Release 释放一个网段中的一个网络
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) (err error) {
	if err = ipam.load(); err != nil {
		return fmt.Errorf("load network info from %s failed, %v", ipam.SubnetAllocatorPath, err)
	}

	c := 0 // 索引位置
	releaseIP := ipaddr.To4()
	releaseIP[3]--
	for t := uint(4); t > 0; t-- {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	if err = ipam.dump(); err != nil {
		return fmt.Errorf("dump network info from %s failed, %v", ipam.SubnetAllocatorPath, err)
	}
	return
}

// load 从本地读取网络信息
func (ipam *IPAM) load() (err error) {
	var (
		subnetConfigFile *os.File
		jsonBytes        []byte
		fStat            os.FileInfo
	)
	if fStat, err = os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			if ipam.Subnets == nil {
				ipam.Subnets = &map[string]string{}
			}
			return nil
		}
		return
	}

	_ = fStat

	if subnetConfigFile, err = os.Open(ipam.SubnetAllocatorPath); err != nil {
		return fmt.Errorf("open file %s failed, %v", ipam.SubnetAllocatorPath, err)
	}
	defer subnetConfigFile.Close()

	if jsonBytes, err = ioutil.ReadAll(subnetConfigFile); err != nil {
		return fmt.Errorf("read file %s content failed, %v", ipam.SubnetAllocatorPath, err)
	}

	dataStr := string(jsonBytes)
	_ = dataStr

	ipam.Subnets = &map[string]string{}
	if err = json.Unmarshal(jsonBytes, ipam.Subnets); err != nil {
		return fmt.Errorf("convert byte to map failed in file %s, %v", ipam.SubnetAllocatorPath, err)
	}

	return
}

// dump 存储网络信息至本地
func (ipam *IPAM) dump() (err error) {
	var (
		dir              = path.Dir(ipam.SubnetAllocatorPath)
		subnetConfigFile *os.File
		jsonBytes        []byte
	)
	if _, err = os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0644); err != nil {
				return fmt.Errorf("create dir %s failed, %v", dir, err)
			}
			return nil
		}
		return err
	}
	if subnetConfigFile, err = os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644); err != nil {
		return fmt.Errorf("open file %s failed, %v", ipam.SubnetAllocatorPath, err)
	}
	defer subnetConfigFile.Close()

	if jsonBytes, err = json.Marshal(ipam.Subnets); err != nil {
		return fmt.Errorf("convert subnet data to json byte failed, %v", err)
	}
	if _, err = subnetConfigFile.Write(jsonBytes); err != nil {
		return fmt.Errorf("write subnet data to file %s failed, %v", ipam.SubnetAllocatorPath, err)
	}
	return
}
