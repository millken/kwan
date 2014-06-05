package core

import (
	"net"
	"sync"
)

func Ip4ToUint32(ip4 net.IP) uint32 {
	return (uint32(ip4[0]) << 24) | (uint32(ip4[1]) << 16) | (uint32(ip4[2]) << 8) | uint32(ip4[3])
}

type PerIpConnTracker struct {
	mutex          sync.Mutex
	perIpConnCount map[uint32]int
}

func (ct *PerIpConnTracker) RegisterIp(ipUint32 uint32) int {
	ct.mutex.Lock()
	ct.perIpConnCount[ipUint32] += 1
	connCount := ct.perIpConnCount[ipUint32]
	ct.mutex.Unlock()
	return connCount
}

func (ct *PerIpConnTracker) UnregisterIp(ipUint32 uint32) {
	ct.mutex.Lock()
	ct.perIpConnCount[ipUint32] -= 1
	ct.mutex.Unlock()
}

func (ct *PerIpConnTracker) GetPerIpConn(ipUint32 uint32) int {
	return ct.perIpConnCount[ipUint32]
}

func (ct *PerIpConnTracker) GetIpConn() map[uint32]int {
	return ct.perIpConnCount
}

func createPerIpConnTracker() *PerIpConnTracker {
	return &PerIpConnTracker{
		perIpConnCount: make(map[uint32]int),
	}
}
