package cnet

import (
	"fmt"
	"testing"
	"time"
)

func TestBridgeNetDriver_Create_Delete(t *testing.T) {
	var (
		n   *Network
		b   = &BridgeNetDriver{}
		err error
	)
	if n, err = b.Create("180.18.0.2/32", "subnet_for_test"); err != nil {
		t.Errorf("create driver failed: %v", err)
	}
	fmt.Println("sleep for 10s, check the result, quickly!")
	time.Sleep(time.Second * 10)

	if err = b.Delete(*n); err != nil {
		t.Errorf("delete driver failed: %v", err)
	}
}
