package utils

import (
	"net"
	"fmt"
	"time"
)

func IpStringToI32(a string) uint32 {
	return IpToI32(net.ParseIP(a))
}

func IpToI32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func I32ToIP(a uint32) net.IP {
	return net.IPv4(byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}


func AddToBlock(ip string, blocktime int) error {

	sock, err := net.DialTimeout("tcp", "127.0.0.1:59101", time.Duration(1) * time.Second)
	if err != nil {
		return err
	}
	defer sock.Close()
	cmd := fmt.Sprintf("add %s %d\n", ip, blocktime)
	sock.Write([]byte(cmd))
	return nil
}