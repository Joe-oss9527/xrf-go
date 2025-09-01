package utils

import (
	"fmt"
	"github.com/fatih/color"
	"time"
)

var (
	Red     = color.New(color.FgRed).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	White   = color.New(color.FgWhite).SprintFunc()

	BoldRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	BoldGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	BoldYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	BoldBlue    = color.New(color.FgBlue, color.Bold).SprintFunc()
	BoldMagenta = color.New(color.FgMagenta, color.Bold).SprintFunc()
	BoldCyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
	BoldWhite   = color.New(color.FgWhite, color.Bold).SprintFunc()
)

func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldGreen("✓"), fmt.Sprintf(format, args...))
}

func PrintError(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldRed("✗"), fmt.Sprintf(format, args...))
}

func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldYellow("⚠"), fmt.Sprintf(format, args...))
}

func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldBlue("ℹ"), fmt.Sprintf(format, args...))
}

func PrintSection(title string) {
	fmt.Println()
	fmt.Println(BoldCyan("━━━ " + title + " ━━━"))
}

func PrintSubSection(title string) {
	fmt.Println(BoldWhite("─── " + title + " ───"))
}

func PrintKeyValue(key, value string) {
	fmt.Printf("  %s: %s\n", BoldWhite(key), value)
}

func PrintProtocolInfo(name, tag, port, status string) {
	statusColor := Green
	if status == "stopped" {
		statusColor = Red
	} else if status == "unknown" {
		statusColor = Yellow
	}

	fmt.Printf("  %s %s [%s] %s\n",
		BoldCyan("•"),
		BoldWhite(name),
		Yellow("Port: "+port),
		statusColor(status))

	if tag != "" {
		fmt.Printf("    Tag: %s\n", tag)
	}
}

func PrintDetailedProtocolInfo(name, tag, protocolType string, port int, settings map[string]interface{}) {
	fmt.Printf("%s %s\n", BoldCyan("━━━"), BoldWhite("协议详细信息"))
	fmt.Printf("  %s: %s\n", BoldWhite("协议名称"), BoldGreen(name))
	fmt.Printf("  %s: %s\n", BoldWhite("标签"), BoldYellow(tag))
	fmt.Printf("  %s: %s\n", BoldWhite("协议类型"), protocolType)
	fmt.Printf("  %s: %d\n", BoldWhite("端口"), port)

	// 显示协议信息
	if desc, exists := settings["description"]; exists {
		fmt.Printf("  %s: %s\n", BoldWhite("描述"), desc)
	}

	if aliases, exists := settings["aliases"]; exists {
		if aliasSlice, ok := aliases.([]string); ok {
			fmt.Printf("  %s: %s\n", BoldWhite("别名"), fmt.Sprintf("[%s]", joinStringSlice(aliasSlice, ", ")))
		}
	}

	// 显示协议特性
	fmt.Printf("\n%s %s\n", BoldCyan("─"), BoldWhite("协议特性"))
	if requiresTLS, exists := settings["requiresTLS"]; exists {
		if requires, ok := requiresTLS.(bool); ok {
			tlsStatus := "否"
			if requires {
				tlsStatus = "是"
			}
			fmt.Printf("  %s: %s\n", BoldWhite("需要 TLS"), tlsStatus)
		}
	}

	if requiresDomain, exists := settings["requiresDomain"]; exists {
		if requires, ok := requiresDomain.(bool); ok {
			domainStatus := "否"
			if requires {
				domainStatus = "是"
			}
			fmt.Printf("  %s: %s\n", BoldWhite("需要域名"), domainStatus)
		}
	}

	if transports, exists := settings["supportedTransports"]; exists {
		if transportSlice, ok := transports.([]string); ok {
			fmt.Printf("  %s: %s\n", BoldWhite("支持的传输方式"), fmt.Sprintf("[%s]", joinStringSlice(transportSlice, ", ")))
		}
	}

	// 显示连接配置
	fmt.Printf("\n%s %s\n", BoldCyan("─"), BoldWhite("连接配置"))

	// 显示 UUID（如果存在）
	if clients, exists := settings["clients"]; exists {
		if clientList, ok := clients.([]interface{}); ok && len(clientList) > 0 {
			if client, ok := clientList[0].(map[string]interface{}); ok {
				if uuid, exists := client["id"]; exists {
					fmt.Printf("  %s: %s\n", BoldWhite("UUID"), uuid)
				}
			}
		}
	}

	// 显示密码（如果存在）
	if method, exists := settings["method"]; exists {
		fmt.Printf("  %s: %s\n", BoldWhite("加密方法"), method)
		if password, exists := settings["password"]; exists {
			fmt.Printf("  %s: %s\n", BoldWhite("密码"), password)
		}
	}

	// 显示传输设置
	if streamSettings, exists := settings["streamSettings"]; exists {
		if streamMap, ok := streamSettings.(map[string]interface{}); ok {
			if network, exists := streamMap["network"]; exists {
				fmt.Printf("  %s: %s\n", BoldWhite("传输协议"), network)
			}

			// WebSocket 设置
			if wsSettings, exists := streamMap["wsSettings"]; exists {
				if wsMap, ok := wsSettings.(map[string]interface{}); ok {
					if path, exists := wsMap["path"]; exists {
						fmt.Printf("  %s: %s\n", BoldWhite("WebSocket 路径"), path)
					}
				}
			}

			// HTTPUpgrade 设置
			if huSettings, exists := streamMap["httpupgradeSettings"]; exists {
				if huMap, ok := huSettings.(map[string]interface{}); ok {
					if path, exists := huMap["path"]; exists {
						fmt.Printf("  %s: %s\n", BoldWhite("HTTPUpgrade 路径"), path)
					}
				}
			}

			// REALITY 设置
			if realitySettings, exists := streamMap["realitySettings"]; exists {
				if realityMap, ok := realitySettings.(map[string]interface{}); ok {
					if dest, exists := realityMap["dest"]; exists {
						fmt.Printf("  %s: %s\n", BoldWhite("目标地址"), dest)
					}
					if serverName, exists := realityMap["serverNames"]; exists {
						if serverNames, ok := serverName.([]interface{}); ok && len(serverNames) > 0 {
							fmt.Printf("  %s: %s\n", BoldWhite("服务器名称"), serverNames[0])
						}
					}
					if shortIds, exists := realityMap["shortIds"]; exists {
						if shortIdList, ok := shortIds.([]interface{}); ok && len(shortIdList) > 0 {
							fmt.Printf("  %s: %s\n", BoldWhite("短 ID"), shortIdList[0])
						}
					}
				}
			}
		}
	}

	fmt.Printf("\n%s %s\n", BoldCyan("─"), BoldWhite("配置文件信息"))
	if configFile, exists := settings["config_file"]; exists {
		fmt.Printf("  %s: %s\n", BoldWhite("配置文件"), configFile)
	}
}

func joinStringSlice(slice []string, sep string) string {
	if len(slice) == 0 {
		return ""
	}
	if len(slice) == 1 {
		return slice[0]
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += sep + slice[i]
	}
	return result
}

func DisableColor() {
	color.NoColor = true
}

func EnableColor() {
	color.NoColor = false
}

// GetCurrentTime 获取当前时间字符串
func GetCurrentTime() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}
