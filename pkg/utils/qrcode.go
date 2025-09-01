package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// GenerateQRCode 生成二维码
func GenerateQRCode(text string) (string, error) {
	// 检查是否安装了 qrencode
	if _, err := exec.LookPath("qrencode"); err == nil {
		return generateQRWithQrencode(text)
	}

	// 如果没有 qrencode，返回 ASCII 艺术二维码提示
	return generateASCIIQR(text), nil
}

// 使用 qrencode 生成二维码
func generateQRWithQrencode(text string) (string, error) {
	cmd := exec.Command("qrencode", "-t", "ANSIUTF8", text)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code with qrencode: %w", err)
	}
	return string(output), nil
}

// 生成 ASCII 艺术二维码（简单版本）
func generateASCIIQR(text string) string {
	// 这是一个简化的 ASCII 二维码表示
	// 在实际应用中，用户应该安装 qrencode 来获得真正的二维码

	qrArt := `
    ████████████████████████████████
    ██                            ██
    ██  扫描二维码或复制链接使用      ██
    ██                            ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██                            ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██  ████  ████  ████  ████    ██
    ██                            ██
    ████████████████████████████████

    注意: 要生成真正的二维码，请安装 qrencode:
    
    Ubuntu/Debian: sudo apt install qrencode
    CentOS/RHEL:   sudo yum install qrencode
    macOS:         brew install qrencode
    
    安装后重新运行此命令可获得扫描二维码。
`

	return qrArt
}

// PrintQRCode 打印二维码和链接
func PrintQRCode(text string, tag string) {
	PrintSection("二维码分享")
	PrintInfo("协议标签: %s", tag)

	qrcode, err := GenerateQRCode(text)
	if err != nil {
		PrintError("生成二维码失败: %v", err)
		PrintSubSection("分享链接")
		fmt.Printf("  %s\n", text)
		return
	}

	// 显示二维码
	fmt.Println(qrcode)

	// 显示链接
	PrintSubSection("分享链接")
	fmt.Printf("  %s\n", text)

	// 显示使用提示
	PrintSubSection("使用说明")
	PrintInfo("1. 使用客户端扫描上方二维码")
	PrintInfo("2. 或复制分享链接手动导入")
	PrintInfo("3. 确保服务器地址正确（使用 --host 参数）")
}

// IsQREncodeAvailable 检查是否安装了 qrencode
func IsQREncodeAvailable() bool {
	_, err := exec.LookPath("qrencode")
	return err == nil
}

// GetQRInstallInstructions 获取 qrencode 安装说明
func GetQRInstallInstructions() string {
	instructions := `
二维码功能需要安装 qrencode 工具：

Ubuntu/Debian:
  sudo apt update && sudo apt install qrencode

CentOS/RHEL/Fedora:
  sudo yum install qrencode
  # 或者: sudo dnf install qrencode

macOS (使用 Homebrew):
  brew install qrencode

Arch Linux:
  sudo pacman -S qrencode

安装完成后，重新运行命令即可显示真正的二维码。
`
	return strings.TrimSpace(instructions)
}
