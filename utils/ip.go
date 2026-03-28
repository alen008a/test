package utils

import (
	"net"
)

//获取本机ip
func GetLocalIP() string {
	addr, _ := net.InterfaceAddrs()
	for _, address := range addr {
		if ip, ok := address.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil {
				return ip.IP.String()
			}
		}
	}
	return ""
}
