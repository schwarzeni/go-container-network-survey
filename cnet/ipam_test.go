package cnet

import (
	"net"
	"testing"
)

func TestIPAM_Allocate(t *testing.T) {
	var ip net.IP
	var err error
	var testInfo = []struct {
		Subnet   string
		TotalNum int
	}{
		{Subnet: "192.168.0.0/24", TotalNum: 255},
		{Subnet: "192.168.0.0/22", TotalNum: 255 * 4},
	}

	for _, d := range testInfo {
		_, ipnet, _ := net.ParseCIDR(d.Subnet)
		ipAllocator.Subnets = &map[string]string{}
		for idx := 0; idx < d.TotalNum; idx++ {
			_, ipnet, _ := net.ParseCIDR(d.Subnet)
			if ip, err = ipAllocator.Allocate(ipnet); err != nil {
				t.Errorf("error with %v, [%d]", err, idx+1)
			}
			t.Log(ip)
		}
		if ip, err = ipAllocator.Allocate(ipnet); err == nil {
			t.Errorf("ip should be used up %s", ip)
		}
	}
}

func TestIPAM_Release(t *testing.T) {
	var ip net.IP
	var err error
	var testInfo = []struct {
		Subnet   string
		TotalNum int
	}{
		{Subnet: "192.168.0.0/24", TotalNum: 255},
		{Subnet: "192.168.0.0/22", TotalNum: 255 * 4},
	}

	for _, d := range testInfo {
		var record []net.IP
		ipAllocator.Subnets = &map[string]string{}
		for idx := 0; idx < d.TotalNum; idx++ {
			_, ipnet, _ := net.ParseCIDR(d.Subnet)
			if ip, err = ipAllocator.Allocate(ipnet); err != nil {
				t.Errorf("error with %v, [%d]", err, idx+1)
			}
			record = append(record, ip)
		}
		for _, ip := range record {
			_, ipnet, _ := net.ParseCIDR(d.Subnet)
			if err = ipAllocator.Release(ipnet, &ip); err != nil {
				t.Errorf("error with %v, [%s]", err, ip.String())
			}
		}
	}
}

func TestReadFile_WriteFile(t *testing.T) {
	var err error
	var record []net.IP
	var ip net.IP
	var testInfo = []struct {
		Subnet   string
		TotalNum int
	}{
		{Subnet: "192.168.0.0/24", TotalNum: 255},
		// {Subnet: "192.168.0.0/22", TotalNum: 255 * 4},
	}

	for _, d := range testInfo {
		for idx := 0; idx < d.TotalNum; idx++ {
			_, ipnet, _ := net.ParseCIDR(d.Subnet)
			if ip, err = ipAllocator.Allocate(ipnet); err != nil {
				t.Errorf("error with %v, [%d]", err, idx+1)
			}
			record = append(record, ip)
		}
	}

	for _, d := range testInfo {
		for idx := 0; idx < d.TotalNum; idx++ {
			_, ipnet, _ := net.ParseCIDR(d.Subnet)
			if err = ipAllocator.Release(ipnet, &record[idx]); err != nil {
				t.Errorf("error with %v, [%d]", err, idx+1)
			}
		}
	}
}
