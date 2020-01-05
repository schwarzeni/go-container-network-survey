# 创建两个网络命名空间
sudo ip netns add ns_py

# 创建一对veth：veth_py_bridge、veth_py_ns
sudo ip link add veth_py_bridge type veth peer name veth_py_ns

# 将一个veth（veth_py_ns）移至命名空间 ns_py 中
sudo ip link set veth_py_ns netns ns_py

# 设置网络命名空间 ns_py 中的设备
sudo ip netns exec ns_py ifconfig veth_py_ns 175.18.0.2/24 up
sudo ip netns exec ns_py route add default dev veth_py_ns

# 创建一个网桥
sudo brctl addbr br_py

# 挂载网络设备
sudo brctl addif br_py veth_py_bridge
# sudo brctl addif br_py enp0s5 # 宿主机默认网卡

# 设置宿主机的路由表
sudo ip link set veth_py_bridge up
sudo ip link set br_py up
sudo route add -net 175.18.0.0/24 dev br_py

# 配置iptables
sudo iptables -t nat -A POSTROUTING -s 175.18.0.0/24 ! -o br_py -j MASQUERADE
sudo iptables -t nat -A PREROUTING -p tcp -m tcp --dport 8089 -j DNAT --to-destination 175.18.0.2:80

