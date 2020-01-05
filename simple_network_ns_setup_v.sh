# ns1
sudo ip netns add ns1
sudo ip link add veth0 type veth peer name veth_ns_1
sudo ip link set veth0 netns ns1
sudo ip netns exec ns1 ifconfig veth0 175.18.0.2/24 up
sudo ip netns exec ns1 ifconfig lo up
sudo ifconfig veth_ns_1 up

# ns2
sudo ip netns add ns2
sudo ip link add veth0 type veth peer name veth_ns_2
sudo ip link set veth0 netns ns2
sudo ip netns exec ns2 ifconfig veth0 175.18.0.3/24 up
sudo ip netns exec ns2 ifconfig lo up
sudo ifconfig veth_ns_2 up

# linux bridge
sudo brctl addbr ns_br
sudo ifconfig ns_br 175.18.0.1/24 up
sudo route add -net 175.18.0.0/24 dev ns_br
sudo brctl addif ns_br veth_ns_1
sudo brctl addif ns_br veth_ns_2
sudo ip netns exec ns1 ip route add default via 175.18.0.1 dev veth0
sudo ip netns exec ns2 ip route add default via 175.18.0.1 dev veth0

# iptables
iptables -t nat -A POSTROUTING -s ns_br ! -o ns_br -j MASQUERADE
iptables -t nat -A PREROUTING -p tcp --dport 8088 -j DNAT --to 175.18.0.2:80
iptables -t nat -A PREROUTING -p tcp --dport 8089 -j DNAT --to 175.18.0.3:80
iptables -t filter -A FORWARD -i ns_br -o ns_br -j ACCEPT
iptables -t filter -A FORWARD -i ns_br ! -o ns_br -j ACCEPT
iptables -t filter -A FORWARD -d 175.18.0.2/32 ! -i ns_br -o ns_br -p tcp -m tcp --dport 80 -j ACCEPT
iptables -t filter -A FORWARD -d 175.18.0.3/32 ! -i ns_br -o ns_br -p tcp -m tcp --dport 80 -j ACCEPT
# optional for ping : iptables -t filter -A FORWARD -o ns_br -j ACCEPT
