package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/xrf-go/pkg/config"
	"github.com/yourusername/xrf-go/pkg/system"
	"github.com/yourusername/xrf-go/pkg/utils"
)

var (
	confDir   string
	verbose   bool
	noColor   bool
	configMgr *config.ConfigManager
	detector  *system.Detector
	installer *system.Installer
	serviceMgr *system.ServiceManager
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "xrf",
		Short: "XRF-Go: 简洁高效的 Xray 安装配置工具",
		Long: `XRF-Go 是一个简洁高效的 Xray 安装配置工具，设计理念为"高效率，超快速，极易用"。
该工具专注核心功能，避免过度工程化，以多配置同时运行为核心设计。

支持的协议:
  • VLESS-REALITY (vr)    - 推荐
  • VLESS-WebSocket-TLS (vw)
  • VMess-WebSocket-TLS (vmess)
  • VLESS-HTTPUpgrade (hu)
  • Trojan-WebSocket-TLS (tw)
  • Shadowsocks (ss)
  • Shadowsocks-2022 (ss2022)`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				utils.SetLogLevel(utils.DEBUG)
			}
			if noColor {
				utils.DisableColor()
			}
			
			// 初始化系统组件
			detector = system.NewDetector()
			installer = system.NewInstaller(detector)
			installer.SetVerbose(verbose)
			serviceMgr = system.NewServiceManager(detector)
			
			// 初始化配置管理器
			configMgr = config.NewConfigManager(confDir)
		},
	}

	// 全局选项
	rootCmd.PersistentFlags().StringVar(&confDir, "confdir", "/etc/xray/confs", "Xray 配置目录")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "禁用彩色输出")

	// 添加所有子命令
	rootCmd.AddCommand(
		createInstallCommand(),
		createAddCommand(),
		createListCommand(),
		createRemoveCommand(),
		createInfoCommand(),
		createChangeCommand(),
		createGenerateCommand(),
		createStartCommand(),
		createStopCommand(),
		createRestartCommand(),
		createStatusCommand(),
		createReloadCommand(),
		createTestCommand(),
		createCheckPortCommand(),
		createBackupCommand(),
		createRestoreCommand(),
		createURLCommand(),
		createQRCommand(),
		createLogsCommand(),
		createVersionCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		utils.PrintError("Error: %v", err)
		os.Exit(1)
	}
}

// 日志查看相关辅助函数

func showLogFile(logFile string, lines int, errorOnly bool) error {
	var cmd *exec.Cmd
	
	if errorOnly {
		// 使用 grep 过滤错误日志
		cmd = exec.Command("sh", "-c", fmt.Sprintf("tail -n %d %s | grep -i 'error\\|failed\\|exception'", lines, logFile))
	} else {
		cmd = exec.Command("tail", "-n", strconv.Itoa(lines), logFile)
	}
	
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}
	
	if len(output) == 0 {
		utils.PrintInfo("没有找到日志内容")
		return nil
	}
	
	fmt.Print(string(output))
	return nil
}

func followLogFile(logFile string, errorOnly bool) error {
	var cmd *exec.Cmd
	
	if errorOnly {
		// 使用 tail -f 跟踪日志并过滤错误
		cmd = exec.Command("sh", "-c", fmt.Sprintf("tail -f %s | grep --line-buffered -i 'error\\|failed\\|exception'", logFile))
	} else {
		cmd = exec.Command("tail", "-f", logFile)
	}
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	utils.PrintInfo("正在跟踪日志文件，按 Ctrl+C 停止...")
	return cmd.Run()
}

func showSystemdJournal(lines int, errorOnly bool) error {
	utils.PrintInfo("使用 systemd journal 查看日志")
	
	args := []string{"-u", "xray", "-n", strconv.Itoa(lines), "--no-pager"}
	if errorOnly {
		args = append(args, "-p", "err")
	}
	
	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		utils.PrintWarning("无法读取 systemd journal: %v", err)
		utils.PrintInfo("请检查 Xray 服务状态：systemctl status xray")
		return nil
	}
	
	return nil
}

func showSystemdJournalFollow() error {
	utils.PrintInfo("正在跟踪 systemd journal，按 Ctrl+C 停止...")
	
	cmd := exec.Command("journalctl", "-u", "xray", "-f", "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		utils.PrintWarning("无法跟踪 systemd journal: %v", err)
		utils.PrintInfo("请检查 Xray 服务状态：systemctl status xray")
		return nil
	}
	
	return nil
}

func createInstallCommand() *cobra.Command {
	var (
		protocols []string
		domain    string
		port      int
		enableBBR bool
		autoFW    bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "一键安装 Xray 服务",
		Long: `一键安装 Xray 服务，自动检测系统并配置服务。

示例:
  xrf install                                    # 默认安装 VLESS-REALITY
  xrf install --protocol vless-reality           # 指定协议
  xrf install --domain example.com --protocols vw,tw  # 多协议安装`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("XRF-Go 安装程序")
			
			// 检查系统支持
			if supported, reason := detector.IsSupported(); !supported {
				return fmt.Errorf("系统不支持: %s", reason)
			}
			
			// 显示系统信息
			if verbose {
				detector.PrintSystemInfo()
			}
			
			// 安装 Xray
			utils.PrintInfo("正在安装 Xray...")
			if err := installer.InstallXray(); err != nil {
				return fmt.Errorf("Xray 安装失败: %w", err)
			}
			
			// 安装并启动服务
			utils.PrintInfo("配置 Xray 服务...")
			if err := serviceMgr.InstallService(); err != nil {
				return fmt.Errorf("服务安装失败: %w", err)
			}
			
			// 初始化配置管理器
			utils.PrintInfo("初始化配置...")
			if err := configMgr.Initialize(); err != nil {
				return fmt.Errorf("配置初始化失败: %w", err)
			}
			
			// 添加指定的协议
			if len(protocols) == 0 {
				protocols = []string{"vless-reality"}
			}
			
			for i, protocolType := range protocols {
				utils.PrintInfo("添加协议 %d/%d: %s", i+1, len(protocols), protocolType)
				
				options := make(map[string]interface{})
				if domain != "" {
					options["domain"] = domain
					options["host"] = domain
				}
				if port != 0 {
					options["port"] = port + i // 为多协议分配不同端口
				}
				
				tag := fmt.Sprintf("%s_%d", strings.ReplaceAll(protocolType, "-", "_"), i+1)
				if len(protocols) == 1 {
					tag = strings.ReplaceAll(protocolType, "-", "_")
				}
				
				if err := configMgr.AddProtocol(protocolType, tag, options); err != nil {
					utils.PrintWarning("添加协议 %s 失败: %v", protocolType, err)
					continue
				}
				
				utils.PrintSuccess("协议 %s 添加成功", protocolType)
			}
			
			// 验证配置
			utils.PrintInfo("验证配置...")
			if err := serviceMgr.ValidateConfig(); err != nil {
				return fmt.Errorf("配置验证失败: %w", err)
			}
			
			// 启动服务
			utils.PrintInfo("启动 Xray 服务...")
			if err := serviceMgr.StartService(); err != nil {
				return fmt.Errorf("启动服务失败: %w", err)
			}
			
			utils.PrintSuccess("🎉 XRF-Go 安装完成!")
			utils.PrintInfo("🔧 管理命令:")
			utils.PrintInfo("  xrf list                 # 查看协议列表")
			utils.PrintInfo("  xrf add [protocol]       # 添加新协议")
			utils.PrintInfo("  xrf status               # 查看服务状态")
			utils.PrintInfo("  xrf logs                 # 查看运行日志")
			
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&protocols, "protocols", "p", nil, "要安装的协议列表")
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "域名")
	cmd.Flags().IntVar(&port, "port", 0, "端口")
	cmd.Flags().BoolVar(&enableBBR, "enable-bbr", true, "启用 BBR 拥塞控制")
	cmd.Flags().BoolVar(&autoFW, "auto-firewall", true, "自动配置防火墙")

	return cmd
}

func createAddCommand() *cobra.Command {
	var (
		port     int
		domain   string
		path     string
		password string
		uuid     string
		tag      string
		noReload bool
	)

	cmd := &cobra.Command{
		Use:   "add [protocol]",
		Short: "添加协议配置",
		Long: `添加新的协议配置到 Xray 服务。

支持的协议别名:
  • vr        - VLESS-REALITY
  • vw        - VLESS-WebSocket-TLS
  • vmess/mw  - VMess-WebSocket-TLS
  • tw        - Trojan-WebSocket-TLS
  • ss        - Shadowsocks
  • ss2022    - Shadowsocks-2022
  • hu        - VLESS-HTTPUpgrade

示例:
  xrf add vr --port 443 --domain example.com    # 添加 VLESS-REALITY
  xrf add vmess --port 80 --path /ws            # 添加 VMess-WebSocket
  xrf add ss --method aes-256-gcm               # 添加 Shadowsocks`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			protocolType := args[0]
			
			utils.PrintSection("添加协议配置")
			utils.PrintInfo("协议: %s", protocolType)
			
			// 构建选项
			options := make(map[string]interface{})
			if port != 0 {
				options["port"] = port
			}
			if domain != "" {
				options["domain"] = domain
				options["host"] = domain
			}
			if path != "" {
				options["path"] = path
			}
			if password != "" {
				options["password"] = password
			}
			if uuid != "" {
				options["uuid"] = uuid
			}
			
			// 生成 tag
			if tag == "" {
				protocol, err := config.DefaultProtocolManager.GetProtocol(protocolType)
				if err != nil {
					return err
				}
				tag = strings.ToLower(strings.ReplaceAll(protocol.Name, "-", "_"))
			}
			
			// 添加协议
			if err := configMgr.AddProtocol(protocolType, tag, options); err != nil {
				return fmt.Errorf("添加协议失败: %w", err)
			}
			
			utils.PrintSuccess("协议 %s 添加成功，标签: %s", protocolType, tag)
			
			// 显示配置信息
			utils.PrintSubSection("配置信息")
			if port != 0 {
				utils.PrintKeyValue("端口", strconv.Itoa(port))
			}
			if domain != "" {
				utils.PrintKeyValue("域名", domain)
			}
			if path != "" {
				utils.PrintKeyValue("路径", path)
			}
			
			// 自动热重载配置
			if !noReload {
				utils.PrintInfo("自动热重载配置...")
				if err := configMgr.ReloadConfig(); err != nil {
					utils.PrintWarning("热重载失败: %v", err)
					utils.PrintInfo("请手动执行 'xrf reload' 重载配置")
				}
			}
			
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "端口")
	cmd.Flags().StringVar(&domain, "domain", "", "域名")
	cmd.Flags().StringVar(&path, "path", "", "路径")
	cmd.Flags().StringVar(&password, "password", "", "密码")
	cmd.Flags().StringVar(&uuid, "uuid", "", "UUID")
	cmd.Flags().StringVar(&tag, "tag", "", "配置标签")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "跳过自动热重载")

	return cmd
}

func createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有协议配置",
		Long:  `列出当前配置的所有协议及其状态。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("协议配置列表")
			
			protocols, err := configMgr.ListProtocols()
			if err != nil {
				return fmt.Errorf("获取协议列表失败: %w", err)
			}
			
			if len(protocols) == 0 {
				utils.PrintInfo("没有找到协议配置")
				return nil
			}
			
			for _, protocol := range protocols {
				status := "运行中"
				if protocol.Status == "stopped" {
					status = "已停止"
				} else if protocol.Status == "unknown" {
					status = "未知"
				}
				
				utils.PrintProtocolInfo(
					protocol.Type,
					protocol.Tag,
					strconv.Itoa(protocol.Port),
					status,
				)
			}
			
			utils.PrintInfo("\n总计: %d 个协议配置", len(protocols))
			return nil
		},
	}

	return cmd
}

func createRemoveCommand() *cobra.Command {
	var noReload bool
	
	cmd := &cobra.Command{
		Use:   "remove [tag]",
		Short: "删除协议配置",
		Long:  `根据标签删除指定的协议配置。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			
			utils.PrintSection("删除协议配置")
			utils.PrintInfo("标签: %s", tag)
			
			if err := configMgr.RemoveProtocol(tag); err != nil {
				return fmt.Errorf("删除协议失败: %w", err)
			}
			
			utils.PrintSuccess("协议配置 %s 删除成功", tag)
			
			// 自动热重载配置
			if !noReload {
				utils.PrintInfo("自动热重载配置...")
				if err := configMgr.ReloadConfig(); err != nil {
					utils.PrintWarning("热重载失败: %v", err)
					utils.PrintInfo("请手动执行 'xrf reload' 重载配置")
				}
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "跳过自动热重载")
	return cmd
}

func createInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [tag]",
		Short: "查看协议配置详情",
		Long:  `查看指定协议的详细配置信息。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			
			// 获取协议详细信息
			info, err := configMgr.GetProtocolInfo(tag)
			if err != nil {
				return fmt.Errorf("获取协议信息失败: %w", err)
			}
			
			// 添加配置文件信息到 settings
			info.Settings["config_file"] = info.ConfigFile
			
			// 显示详细信息
			utils.PrintDetailedProtocolInfo(info.Type, info.Tag, info.Type, info.Port, info.Settings)
			
			return nil
		},
	}

	return cmd
}

func createChangeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change [tag] [key] [value]",
		Short: "修改协议配置",
		Long: `修改指定协议的配置参数。

支持的参数:
  • port       - 端口号 (数字)
  • password   - 密码 (字符串)
  • uuid       - UUID (字符串)
  • path       - 路径 (字符串)

示例:
  xrf change vless_reality port 8443     # 修改端口
  xrf change trojan_ws password newpass  # 修改密码
  xrf change vmess_ws uuid new-uuid      # 修改 UUID
  xrf change vless_ws path /newpath      # 修改路径`,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, key, value := args[0], args[1], args[2]
			
			utils.PrintSection("修改协议配置")
			utils.PrintInfo("标签: %s", tag)
			utils.PrintInfo("参数: %s -> %s", key, value)
			
			// 验证配置是否存在
			_, err := configMgr.GetProtocolInfo(tag)
			if err != nil {
				return fmt.Errorf("协议配置不存在: %w", err)
			}
			
			// 转换值类型
			var typedValue interface{} = value
			switch key {
			case "port":
				if portInt, err := strconv.Atoi(value); err != nil {
					return fmt.Errorf("端口必须是数字: %s", value)
				} else if portInt < 1 || portInt > 65535 {
					return fmt.Errorf("端口必须在 1-65535 范围内")
				} else {
					typedValue = portInt
				}
			case "uuid":
				if !utils.IsValidUUID(value) {
					return fmt.Errorf("无效的 UUID 格式: %s", value)
				}
			case "path":
				if !strings.HasPrefix(value, "/") {
					return fmt.Errorf("路径必须以 / 开头: %s", value)
				}
			case "password":
				if len(value) < 6 {
					return fmt.Errorf("密码长度至少 6 位")
				}
			}
			
			options := map[string]interface{}{
				key: typedValue,
			}
			
			if err := configMgr.UpdateProtocol(tag, options); err != nil {
				return fmt.Errorf("修改协议配置失败: %w", err)
			}
			
			utils.PrintSuccess("协议配置修改成功")
			
			// 显示修改后的信息
			if key == "port" || key == "password" || key == "uuid" || key == "path" {
				utils.PrintInfo("新值: %v", typedValue)
			}
			
			return nil
		},
	}

	return cmd
}

func createGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [type]",
		Short: "生成密码、UUID、密钥等",
		Long: `生成各种类型的密码、UUID、密钥。

支持的类型:
  • password  - 随机密码
  • uuid      - UUID v4
  • ss2022    - Shadowsocks 2022 密钥
  • keypair   - X25519 密钥对
  • shortid   - REALITY 短 ID`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			genType := args[0]
			
			utils.PrintSection("生成工具")
			
			switch strings.ToLower(genType) {
			case "password":
				password := utils.GeneratePassword(16)
				utils.PrintKeyValue("随机密码", password)
				
			case "uuid":
				uuid := utils.GenerateUUID()
				utils.PrintKeyValue("UUID", uuid)
				
			case "ss2022":
				key, err := utils.GenerateShadowsocks2022Key("2022-blake3-aes-256-gcm")
				if err != nil {
					return fmt.Errorf("生成 Shadowsocks 2022 密钥失败: %w", err)
				}
				utils.PrintKeyValue("SS2022 密钥", key)
				
			case "keypair", "pbk":
				priv, pub, err := utils.GenerateX25519KeyPair()
				if err != nil {
					return fmt.Errorf("生成密钥对失败: %w", err)
				}
				utils.PrintKeyValue("私钥", priv)
				utils.PrintKeyValue("公钥", pub)
				
			case "shortid":
				shortId := utils.GenerateShortID(8)
				utils.PrintKeyValue("短 ID", shortId)
				
			default:
				return fmt.Errorf("不支持的生成类型: %s", genType)
			}
			
			return nil
		},
	}

	return cmd
}

// 服务管理命令
func createStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "启动 Xray 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceMgr.StartService()
		},
	}
}

func createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止 Xray 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceMgr.StopService()
		},
	}
}

func createRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重启 Xray 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceMgr.RestartService()
		},
	}
}

func createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看服务状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceMgr.PrintServiceStatus()
		},
	}
}

func createReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "热重载配置",
		Long: `热重载 Xray 配置文件，无需重启服务。

该命令会：
1. 验证配置文件的正确性
2. 向运行中的 Xray 进程发送 USR1 信号
3. Xray 自动重新加载配置

注意: 仅对配置文件的修改生效，不会重新加载二进制文件或系统服务配置。

示例:
  xrf reload    # 热重载当前配置`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return configMgr.ReloadConfig()
		},
	}
}

func createTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "验证配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serviceMgr.ValidateConfig()
		},
	}
}

func createCheckPortCommand() *cobra.Command {
	var (
		checkRange string
		protocol   string
		suggest    bool
	)
	
	cmd := &cobra.Command{
		Use:   "check-port [port]",
		Short: "检查端口可用性",
		Long: `检查指定端口是否可用，支持端口范围检查和协议建议。

示例:
  xrf check-port 443                    # 检查单个端口
  xrf check-port --range 8000-9000      # 检查端口范围
  xrf check-port --protocol vless-reality --suggest  # 获取协议端口建议`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("端口检查")
			
			if suggest && protocol != "" {
				// 获取协议建议端口
				suggestedPorts := utils.GetPortsByProtocol(protocol)
				utils.PrintInfo("协议 %s 推荐端口:", protocol)
				
				availablePorts := []int{}
				for _, port := range suggestedPorts {
					if utils.IsPortAvailable(port) {
						availablePorts = append(availablePorts, port)
						fmt.Printf("  %s %d - 可用\n", utils.BoldGreen("✓"), port)
					} else {
						fmt.Printf("  %s %d - 已占用\n", utils.BoldRed("✗"), port)
					}
				}
				
				if len(availablePorts) > 0 {
					utils.PrintSuccess("建议使用端口: %d", availablePorts[0])
				} else {
					utils.PrintWarning("所有推荐端口均已占用，寻找替代端口...")
					if altPort, err := utils.SuggestPort(protocol, 0); err == nil {
						utils.PrintSuccess("建议替代端口: %d", altPort)
					} else {
						utils.PrintError("无法找到可用端口: %v", err)
					}
				}
				return nil
			}
			
			if checkRange != "" {
				// 检查端口范围
				parts := strings.Split(checkRange, "-")
				if len(parts) != 2 {
					return fmt.Errorf("端口范围格式错误，应为: start-end")
				}
				
				startPort, err := strconv.Atoi(parts[0])
				if err != nil {
					return fmt.Errorf("起始端口无效: %s", parts[0])
				}
				
				endPort, err := strconv.Atoi(parts[1])
				if err != nil {
					return fmt.Errorf("结束端口无效: %s", parts[1])
				}
				
				utils.PrintInfo("检查端口范围: %d-%d", startPort, endPort)
				
				availableCount := 0
				for port := startPort; port <= endPort; port++ {
					if utils.IsPortAvailable(port) {
						availableCount++
					}
				}
				
				totalPorts := endPort - startPort + 1
				utils.PrintInfo("总端口数: %d", totalPorts)
				utils.PrintInfo("可用端口数: %d", availableCount)
				utils.PrintInfo("已占用端口数: %d", totalPorts - availableCount)
				
				if availableCount > 0 {
					if availablePort, err := utils.FindAvailablePort(startPort, endPort); err == nil {
						utils.PrintSuccess("第一个可用端口: %d", availablePort)
					}
				}
				
				return nil
			}
			
			if len(args) == 0 {
				return fmt.Errorf("请指定要检查的端口或使用 --range 参数")
			}
			
			// 检查单个端口
			port, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("端口格式错误: %s", args[0])
			}
			
			if port < 1 || port > 65535 {
				return fmt.Errorf("端口范围必须在 1-65535 之间")
			}
			
			utils.PrintInfo("检查端口: %d", port)
			
			if utils.IsPortAvailable(port) {
				utils.PrintSuccess("端口 %d 可用", port)
			} else {
				utils.PrintError("端口 %d 已被占用", port)
			}
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&checkRange, "range", "", "检查端口范围 (格式: start-end)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "协议类型 (配合 --suggest 使用)")
	cmd.Flags().BoolVar(&suggest, "suggest", false, "获取协议端口建议")
	
	return cmd
}

func createBackupCommand() *cobra.Command {
	var backupPath string
	
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "备份配置",
		Long: `备份当前的协议配置到压缩文件。

如果不指定输出路径，将生成默认的时间戳文件名。

示例:
  xrf backup                           # 备份到默认文件
  xrf backup --output my-backup.tar.gz # 备份到指定文件`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("配置备份")
			
			if backupPath != "" {
				utils.PrintInfo("备份路径: %s", backupPath)
			} else {
				utils.PrintInfo("使用默认备份路径")
			}
			
			if err := configMgr.BackupConfig(backupPath); err != nil {
				return fmt.Errorf("备份失败: %w", err)
			}
			
			utils.PrintSuccess("配置备份完成")
			return nil
		},
	}
	
	cmd.Flags().StringVarP(&backupPath, "output", "o", "", "备份文件路径（可选）")
	return cmd
}

func createRestoreCommand() *cobra.Command {
	var confirmRestore bool
	
	cmd := &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "恢复配置",
		Long: `从备份文件恢复配置。

警告: 此操作将替换当前的所有配置。建议先备份当前配置。

示例:
  xrf restore my-backup.tar.gz --confirm    # 恢复配置`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupFile := args[0]
			
			utils.PrintSection("配置恢复")
			utils.PrintInfo("备份文件: %s", backupFile)
			
			if !confirmRestore {
				utils.PrintWarning("此操作将替换当前的所有配置！")
				utils.PrintWarning("建议先执行 'xrf backup' 备份当前配置")
				utils.PrintError("请使用 --confirm 参数确认执行恢复操作")
				return fmt.Errorf("恢复操作需要确认")
			}
			
			utils.PrintInfo("正在恢复配置...")
			
			if err := configMgr.RestoreConfig(backupFile); err != nil {
				return fmt.Errorf("恢复失败: %w", err)
			}
			
			utils.PrintSuccess("配置恢复完成")
			utils.PrintInfo("建议执行 'xrf test' 验证配置")
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&confirmRestore, "confirm", false, "确认执行恢复操作")
	return cmd
}

func createURLCommand() *cobra.Command {
	var showHost bool
	var customHost string
	
	cmd := &cobra.Command{
		Use:   "url [tag]",
		Short: "生成分享链接",
		Long: `生成指定协议的分享链接，支持各种客户端格式。

注意: 默认使用 localhost 作为主机地址，请使用 --host 参数指定实际的服务器地址。

示例:
  xrf url vless_reality --host example.com    # 生成 VLESS-REALITY 链接
  xrf url vmess_ws --host 192.168.1.100       # 生成 VMess 链接
  xrf url --list                              # 显示所有可用的协议标签`,
		Args: func(cmd *cobra.Command, args []string) error {
			if showHost {
				return nil // --list 不需要参数
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showHost {
				// 显示所有可用的协议
				protocols, err := configMgr.ListProtocols()
				if err != nil {
					return fmt.Errorf("获取协议列表失败: %w", err)
				}
				
				utils.PrintSection("可用的协议配置")
				for _, protocol := range protocols {
					utils.PrintInfo("• %s (%s)", protocol.Tag, protocol.Type)
				}
				return nil
			}
			
			tag := args[0]
			
			utils.PrintSection("生成分享链接")
			utils.PrintInfo("协议标签: %s", tag)
			
			// 生成分享链接
			shareURL, err := configMgr.GenerateShareURL(tag)
			if err != nil {
				return fmt.Errorf("生成分享链接失败: %w", err)
			}
			
			// 如果用户指定了自定义主机，替换 URL 中的主机
			if customHost != "" {
				shareURL = strings.Replace(shareURL, "localhost", customHost, 1)
			}
			
			utils.PrintSubSection("分享链接")
			fmt.Printf("  %s\n", shareURL)
			
			// 显示提示信息
			if customHost == "" {
				utils.PrintWarning("注意: 链接使用 'localhost' 作为主机地址")
				utils.PrintInfo("使用 --host 参数指定实际的服务器地址")
				utils.PrintInfo("例如: xrf url %s --host your-server.com", tag)
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&showHost, "list", false, "显示所有可用的协议标签")
	cmd.Flags().StringVar(&customHost, "host", "", "指定服务器主机地址")
	
	return cmd
}

func createQRCommand() *cobra.Command {
	var customHost string
	
	cmd := &cobra.Command{
		Use:   "qr [tag]",
		Short: "显示二维码",
		Long: `显示指定协议的二维码，方便移动端扫描导入。

注意: 默认使用 localhost 作为主机地址，请使用 --host 参数指定实际的服务器地址。

示例:
  xrf qr vless_reality --host example.com     # 显示 VLESS-REALITY 二维码
  xrf qr vmess_ws --host 192.168.1.100        # 显示 VMess 二维码`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			
			// 生成分享链接
			shareURL, err := configMgr.GenerateShareURL(tag)
			if err != nil {
				return fmt.Errorf("生成分享链接失败: %w", err)
			}
			
			// 如果用户指定了自定义主机，替换 URL 中的主机
			if customHost != "" {
				shareURL = strings.Replace(shareURL, "localhost", customHost, 1)
			}
			
			// 显示二维码
			utils.PrintQRCode(shareURL, tag)
			
			// 显示提示信息
			if customHost == "" {
				utils.PrintWarning("\n注意: 链接使用 'localhost' 作为主机地址")
				utils.PrintInfo("使用 --host 参数指定实际的服务器地址")
				utils.PrintInfo("例如: xrf qr %s --host your-server.com", tag)
			}
			
			// 如果没有安装 qrencode，显示安装说明
			if !utils.IsQREncodeAvailable() {
				utils.PrintSubSection("安装说明")
				fmt.Println(utils.GetQRInstallInstructions())
			}
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&customHost, "host", "", "指定服务器主机地址")
	
	return cmd
}

func createLogsCommand() *cobra.Command {
	var follow bool
	var lines int
	var errorOnly bool
	
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "查看运行日志",
		Long: `查看 Xray 服务的运行日志。

默认显示最新的 50 行日志。使用 -f 参数可以实时跟踪日志。

示例:
  xrf logs                    # 显示最新 50 行日志
  xrf logs -n 100             # 显示最新 100 行日志
  xrf logs -f                 # 实时跟踪日志
  xrf logs --error            # 只显示错误日志`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("Xray 运行日志")
			
			// 常见的日志文件位置
			logPaths := []string{
				"/var/log/xray/access.log",
				"/var/log/xray/error.log",
				"/var/log/xray.log",
				"/tmp/xray.log",
			}
			
			var logFile string
			for _, path := range logPaths {
				if _, err := os.Stat(path); err == nil {
					logFile = path
					break
				}
			}
			
			// 如果找不到日志文件，尝试使用 systemd journal
			if logFile == "" {
				if follow {
					return showSystemdJournalFollow()
				} else {
					return showSystemdJournal(lines, errorOnly)
				}
			}
			
			utils.PrintInfo("日志文件: %s", logFile)
			
			// 显示日志
			if follow {
				return followLogFile(logFile, errorOnly)
			} else {
				return showLogFile(logFile, lines, errorOnly)
			}
		},
	}
	
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "实时跟踪日志")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "显示的行数")
	cmd.Flags().BoolVar(&errorOnly, "error", false, "只显示错误日志")
	
	return cmd
}

func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("版本信息")
			utils.PrintKeyValue("XRF-Go 版本", "v1.0.0-dev")
			utils.PrintKeyValue("构建时间", "未设置")
			utils.PrintKeyValue("Go 版本", "1.23+")
			return nil
		},
	}
}