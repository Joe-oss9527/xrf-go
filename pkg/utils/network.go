package utils

import (
	"fmt"
	"net"
	"strconv"
)

// GetRandomAvailablePort 获取随机可用端口
func GetRandomAvailablePort() (int, error) {
	// 使用系统分配的随机端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// GetUsedPorts 获取已使用的端口列表
func GetUsedPorts() ([]int, error) {
	// 这里实现获取系统已使用端口的逻辑
	// 可以通过解析 /proc/net/tcp 和 /proc/net/udp 文件
	// 或者使用 netstat 命令
	return []int{}, nil // 简化实现
}

// ValidatePortRange 验证端口范围
func ValidatePortRange(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	return port, nil
}
