# 《自己动手写Docker》第六章相关技术调研

容器网络相关技术

---

## 环境参数

- Ubuntu 16.04
- Linux 4.10.0-28-generic x86_64
- go 1.12.4 linux/amd64

---

## 目前存在的问题

- bridge 模式的容器无法在本机通过 `lo` 网络接口访问，举个例子，如果做了端口映射 "8085 --> 80" ，那么无法在本机上用 "127.0.0.1:8085" 来访问容器的网络服务。本实验使用 `iptables` 进行流量转发，网上的[一种说法](https://www.linuxquestions.org/questions/linux-networking-3/iptables-redirect-127-0-0-1-to-192-168-1-113-a-818817/#3)是 iptables 无法直接操作 lo 接口的流量
- 目前仅支持容器应用的前台运行。有本次调研仅仅是为了探索容器 bridge 网络模式的配置，所以就不涉及到管理后台运行容器的相关内容，也为将容器的文件系统挂载到一个新的文件系统中，仅仅是最简单的命名空间隔离。
- 根据网段分发 ip 的算法使用书中的算法，使用字符串而不是整形数字作为 bitmap 数据的存储结构，所以效率较低
- 本样例代码使用 `python3 -m http.server <port>` 作为服务器运行于容器内部，请确保 python3 的可执行文件存放于目录： `/usr/bin/python3`

---

## 可用命令

为了是这部分逻辑简单，没有做任何输入合法性校验，格式也非常严格，下面演示：

1. 创建一个 bridge
2. 运行一个容器，将其挂载在创建的 bridge 上
3. 再运行一个容器，将其挂载在创建的 bridge 上
4. 删除这个 bridge

本样例直接在根目录执行 `go run . arg1 arg2 ...` 运行，本项目使用了一些第三方库，下载某些被墙的依赖时需要配置 `GOPROXY` 环境变量为 `https://goproxy.io` 作为网络代理。

```bash
# 1.创建一个 bridge
# 参数缺一不可，且顺序不能乱
# -cnet：表示创建一个新的 bridge
# mynet1：bridge名称，不能重复
# bridge：使用驱动名称，目前只能写 bridge
# 175.50.0.0/24：创建的 bridge 的网段
go run . -cnet mynet1 bridge 175.50.0.0/24

# 2. 运行一个容器，将其挂载在创建的 bridge 上
# 参数缺一不可，且顺序不能乱
# -run：表示运行一个容器
# 2233：为容器的ID，不可重复
# mynet1：为使用 bridge 的名称
# 8085:80：配置端口映射，8085为主机端口，80为容器网络命名空间中的端口
go run . -run 2233 mynet1 8085:80
```

此时有类似于如下的输出，并阻塞：

```txt
2020/01/10 13:46:36 start container[2233]:
id: 2233
pid: 26000
container ip: 175.50.0.2/24
port mapping: [8085:80]

run this command to enter container:
nsenter --target 26000 --mount --uts --ipc --net --pid

Serving HTTP on 0.0.0.0 port 80 ...
```

在打开另一个终端界面，输入类似前一条命令：

```bash
go run . -run 3344 mynet1 8086:80
```

此时有类似于如下的输出，并阻塞：

```txt
2020/01/10 13:51:18 start container[3344]:
id: 3344
pid: 4439
container ip: 175.50.0.3/24
port mapping: [8086:80]

run this command to enter container:
nsenter --target 4439 --mount --uts --ipc --net --pid

Serving HTTP on 0.0.0.0 port 80 ...
```

假设主机 ip 为 10.211.55.32 ，则此时可以在本机或者其他主机上执行如下命令进行测试：

```bash
curl http://10.211.55.32:8085
curl http://10.211.55.32:8086
```

可以按照提示执行类似于 `nsenter --target 4439 --mount --uts --ipc --net --pid` 的命令进入容器命名空间中查看相关的网络配置，执行 `exit` 退出。

ctrl-c 结束两个容器进程，执行如下的命令删除 bridge：

```bash
# -dnet：表示删除 bridge
# mynet1：指定网桥名称
go run . -dnet mynet1
```

---

## 技术要点

编写代码不难，主要是写代码之前先把流程梳理好，数据结构制定好，iptables 的相关代码先做好测试，之后写出来的代码基本上没问题。

### iptables

iptables 配置有点难，其实现在作者还没搞太懂，主要还是靠着《自己动手写Docker》以及本机上 Docker 对 iptables 的配置照葫芦画瓢写的，在项目根目录下有 [simple_network_ns_setup_v.sh](./simple_network_ns_setup_v.sh) 文件，这个是如何使用命令行创建两个独立的网络命名空间，同时再创建一个 bridge 作为两个网络命名空间和外部通信的桥梁，[static_server.sh](./static_server.sh) 为用 python 来启动一个简答的 web 服务器

iptables 用来对 tcp 数据包进行转发，关于它和网络命名空间的学习笔记参见作者之前写的这篇 [veth + iptables 模拟 Docker 网络 Bridge 模式](https://blog.schwarzeni.com/2019/12/28/veth-iptables-%E6%A8%A1%E6%8B%9F-Docker-%E7%BD%91%E7%BB%9C-Bridge-%E6%A8%A1%E5%BC%8F/)

---

### bridge 创建和删除流程

主要在文件 [cnet/opt.go](./cnet/opt.go) 以及 [cnet/bridge_net_driver.go](./cnet/bridge_net_driver.go) 中

创建 bridge 的流程

1. 根据 ip 范围分配网关的 ip 地址
2. 创建Bridge虚拟设备
3. 设置Bridge设备的地址和路由
4. 启动Bridge设备
5. 设置iptables的规则

删除 bridge 的流程

1. 释放网关的 ip 地址
2. 删除 bridge 网络设备
3. 删除 iptables 的相关规则

---

### 进入指定网络命名空间

这里由于需要进入到容器的网络命名空间中进行相关的网络配置，所以使用到了第三方库的相关方法，根据容器进程的 PID 进入到指定网络命名空间中

[cnet/net_utils.go](./cnet/net_utils.go)

```go
import (
  "github.com/vishvananda/netlink"
  "github.com/vishvananda/netns"
)

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
```

调用该函数进行一些配置: [cnet/net_utils.go](./cnet/net_utils.go)

```go
// configEndpointIPAddressAndRoute 容器内部配置容器网络、设备 IP 地址和路由信息
func configEndpointIPAddressAndRoute(ep *EndPoint, cinfo *container.Info) (err error) {
	var (
		peerLink netlink.Link
	)
	if peerLink, err = netlink.LinkByName(ep.Device.PeerName); err != nil {
		return fmt.Errorf("get interface %s failed, %v", ep.Device.PeerName, err)
	}

  defer enterContainerNetns(&peerLink, cinfo)()

  // ...

	return
}
```

其中，`defer` 会先执行函数 `enterContainerNetns` 中的内容，在函数 `configEndpointIPAddressAndRoute` 调用结束后，在执行回调函数的内容，如下，确保正确退出当前的网络命名空间

```go
func() {
		_ = netns.Set(origns)
		_ = origns.Close()
		runtime.UnlockOSThread()
		_ = f.Close()
	}
```

---

当然还有一些小技巧，比如使用在 [main.go](./main.go) 中使用 Linux 管道来传递数据、同步消息，相关的技巧在作者之前的文章 [Linux 管道代码样例(一)](https://blog.schwarzeni.com/2019/12/22/Linux-%E7%AE%A1%E9%81%93%E4%BB%A3%E7%A0%81%E6%A0%B7%E4%BE%8B-%E4%B8%80/#Go%E8%AF%AD%E8%A8%80%E4%BD%BF%E7%94%A8%E7%AE%A1%E9%81%93) 中提到过。

再比如如何使用 go 监听相关的 Linux 信号，比如 `SIGTERM`，在 [main.go](./main.go) 就实现了在 ctrl+c 退出的时候只需相关的清理函数，去除容器的相关配置，相关的技巧在作者之前的文章 [Golang优雅地结束server](https://blog.schwarzeni.com/2019/10/20/Golang%E4%BC%98%E9%9B%85%E5%9C%B0%E7%BB%93%E6%9D%9Fserver/#%E8%87%AA%E5%AE%9A%E4%B9%89%E4%BF%A1%E5%8F%B7%E9%87%8F%E7%9B%91%E5%90%AC%E5%99%A8) 提到过类似的实现。
