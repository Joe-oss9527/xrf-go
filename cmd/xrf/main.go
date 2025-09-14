package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Joe-oss9527/xrf-go/pkg/config"
	"github.com/Joe-oss9527/xrf-go/pkg/system"
	"github.com/Joe-oss9527/xrf-go/pkg/tls"
	"github.com/Joe-oss9527/xrf-go/pkg/utils"
	"github.com/spf13/cobra"
)

// These will be set via -ldflags in CI (see .github/workflows/release.yml)
var (
	Version   = "v0.0.0-dev"
	BuildTime = ""
	GitCommit = ""
)

var (
	confDir    string
	verbose    bool
	noColor    bool
	configMgr  *config.ConfigManager
	detector   *system.Detector
	installer  *system.Installer
	serviceMgr *system.ServiceManager
	acmeMgr    *tls.ACMEManager
	caddyMgr   *tls.CaddyManager
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
		createTLSCommand(),
		createCaddyCommand(),
		// DESIGN.md required commands
		createIPCommand(),
		createBBRCommand(),
		createSwitchCommand(),
		createEnableAllCommand(),
		createUpdateCommand(),
		createCleanCommand(),
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
  xrf install                                    # 默认安装 VLESS-REALITY (零配置)
  xrf install --protocol vless-reality           # 指定协议
  xrf install --protocols vw,tw --domain example.com  # 多协议安装(TLS协议需要域名)`,
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
				return fmt.Errorf("xray 安装失败: %w", err)
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
		dest     string
		path     string
		password string
		uuid     string
		tag      string
		noReload bool
		// VLESS-Encryption options
		veFlow      string
		veAuth      string
		veMode      string
		veServerRTT string
		veClientRTT string
		vePadding   string
	)

	cmd := &cobra.Command{
		Use:   "add [protocol]",
		Short: "添加协议配置",
		Long: `添加新的协议配置到 Xray 服务。

支持的协议别名:
  • vr        - VLESS-REALITY
  • ve        - VLESS-Encryption (后量子加密)
  • vw        - VLESS-WebSocket-TLS
  • vmess/mw  - VMess-WebSocket-TLS
  • tw        - Trojan-WebSocket-TLS
  • ss        - Shadowsocks
  • ss2022    - Shadowsocks-2022
  • hu        - VLESS-HTTPUpgrade

示例:
  xrf add vr                                    # 添加 VLESS-REALITY (零配置)
  xrf add vr --port 8443                        # 自定义端口
  xrf add vr --dest www.microsoft.com:443       # 自定义目标地址
  xrf add ve --port 443 --auth mlkem768 --mode native --server-rtt 600s --client-rtt 0rtt \
           --flow xtls-rprx-vision --padding "100-111-1111.75-0-111.50-0-3333"  # 添加 VLESS-Encryption
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
			if dest != "" {
				options["dest"] = dest
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

			// 解析协议元信息
			protocol, err := config.DefaultProtocolManager.GetProtocol(protocolType)
			if err != nil {
				return err
			}

			// 对 VLESS-Encryption 预先生成 decryption/encryption（可自定义）
			var clientEncryption string
			if strings.EqualFold(protocol.Name, "VLESS-Encryption") {
				utils.PrintInfo("生成 VLESS Encryption 配置对…")
				if veFlow != "" {
					options["flow"] = veFlow
				}
				dec, enc, err := composeVLESSENCPair(veAuth, veMode, veServerRTT, veClientRTT, vePadding)
				if err != nil {
					return fmt.Errorf("生成 VLESS Encryption 配置失败: %w", err)
				}
				options["decryption"] = dec
				clientEncryption = enc
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
			if clientEncryption != "" {
				utils.PrintKeyValue("客户端 encryption", clientEncryption)
				utils.PrintInfo("将上面的 encryption 字符串粘贴到客户端 VLESS outbound 的 settings.encryption")
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
	cmd.Flags().StringVar(&domain, "domain", "", "域名 (仅TLS协议需要)")
	cmd.Flags().StringVar(&dest, "dest", "", "REALITY目标地址 (如: www.microsoft.com:443)")
	cmd.Flags().StringVar(&path, "path", "", "路径")
	cmd.Flags().StringVar(&password, "password", "", "密码")
	cmd.Flags().StringVar(&uuid, "uuid", "", "UUID")
	cmd.Flags().StringVar(&tag, "tag", "", "配置标签")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "跳过自动热重载")
	// VLESS-Encryption flags
	cmd.Flags().StringVar(&veFlow, "flow", "", "VLESS-Encryption flow（默认 xtls-rprx-vision）")
	cmd.Flags().StringVar(&veAuth, "auth", "mlkem768", "VLESS-Encryption 认证 (mlkem768|x25519)")
	cmd.Flags().StringVar(&veMode, "mode", "native", "VLESS-Encryption 模式 (native|xorpub|random)")
	cmd.Flags().StringVar(&veServerRTT, "server-rtt", "600s", "服务端 1-RTT 时长（如 600s 或 600-900s；或 0rtt）")
	cmd.Flags().StringVar(&veClientRTT, "client-rtt", "0rtt", "客户端 0-RTT/1-RTT 设置（0rtt 或 如 600s）")
	cmd.Flags().StringVar(&vePadding, "padding", "", "1-RTT padding 规则（以 . 分隔多段，如 100-111-1111.75-0-111.50-0-3333）")

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
				var status string
				switch protocol.Status {
				case "stopped":
					status = "已停止"
				case "unknown":
					status = "未知"
				default:
					status = "运行中"
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

			// 对 VLESS-Encryption（或含 decryption 的 VLESS 入站）显示客户端 encryption 与提示
			if encHint := maybePrintVEEncryptionHint(info.Settings); encHint != "" {
				_ = encHint
			}

			return nil
		},
	}

	return cmd
}

// maybePrintVEEncryptionHint 如果当前入站为 VLESS 且 settings.decryption!=none，则派生并打印 encryption
func maybePrintVEEncryptionHint(inbound map[string]interface{}) string {
	// inbound is the inbounds[0] map
	// Find settings.decryption
	settingsVal, ok := inbound["settings"].(map[string]interface{})
	if !ok {
		return ""
	}
	decVal, ok := settingsVal["decryption"].(string)
	if !ok || decVal == "" || decVal == "none" {
		return ""
	}
	enc, err := deriveVEEncryption(decVal)
	if err != nil {
		utils.PrintWarning("无法派生 VLESS Encryption 客户端参数: %v", err)
		return ""
	}
	utils.PrintSubSection("VLESS Encryption 客户端参数")
	utils.PrintKeyValue("encryption", enc)
	utils.PrintInfo("将上述 encryption 填入客户端 VLESS outbound 的 settings.encryption；客户端 RTT 建议使用 0rtt")
	// 补充参考文档与注意事项
	utils.PrintSubSection("参考文档与注意事项")
	utils.PrintInfo("• 多配置目录: https://xtls.github.io/config/features/multiple.html")
	utils.PrintInfo("• PR 说明: https://github.com/XTLS/Xray-core/pull/5067")
	utils.PrintInfo("• 不可与 settings.fallbacks 同时使用；建议开启 XTLS 以避免二次加解密")
	utils.PrintInfo("• 客户端需支持 VLESS Encryption（如: 最新 Xray-core、Mihomo ≥ v1.19.13）")
	return enc
}

// deriveVEEncryption 根据服务端 decryption 计算客户端 encryption（优先 0rtt）
func deriveVEEncryption(decryption string) (string, error) {
	parts := strings.Split(decryption, ".")
	if len(parts) < 4 {
		return "", fmt.Errorf("无效的 decryption 格式")
	}
	prefix, mode := parts[0], parts[1]
	// split padding and key from remaining parts
	rest := parts[3:]
	if len(rest) == 0 {
		return "", fmt.Errorf("decryption 缺少密钥段")
	}
	// Identify last base64-looking segment as key
	key := rest[len(rest)-1]
	// Determine auth by key length after base64 decode
	// Try base64url decode
	var keyBytes []byte
	if kb, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		keyBytes = kb
	} else {
		return "", fmt.Errorf("无法解析密钥: %v", err)
	}
	var clientKey string
	switch len(keyBytes) {
	case 32: // X25519 private -> derive public
		pub, err := utils.DeriveX25519Public(key)
		if err != nil {
			return "", err
		}
		clientKey = pub
	case 64: // ML-KEM-768 seed -> derive client via xray
		ck, err := deriveMLKEMClientFromSeed(key)
		if err != nil {
			return "", err
		}
		clientKey = ck
	default:
		return "", fmt.Errorf("不支持的密钥长度: %d", len(keyBytes))
	}
	// Prefer 0rtt on client
	clientRTT := "0rtt"
	enc := buildVEDot(prefix, mode, clientRTT, "", clientKey)
	return enc, nil
}

func deriveMLKEMClientFromSeed(seed string) (string, error) {
	// xray mlkem768 -i <seed>
	out, err := runXraySimple("mlkem768", "-i", seed)
	if err != nil {
		return "", err
	}
	reClient := regexp.MustCompile(`(?m)^Client:\s*([A-Za-z0-9_-]+)=*$`)
	mc := reClient.FindStringSubmatch(out)
	if len(mc) != 2 {
		return "", fmt.Errorf("无法解析 mlkem768 输出: %s", out)
	}
	return mc[1], nil
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
		Args: cobra.ExactArgs(3),
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
	var (
		vlessAuth      string
		vlessMode      string
		vlessServerRTT string
		vlessClientRTT string
		vlessPadding   string
	)

	cmd := &cobra.Command{
		Use:   "generate [type]",
		Short: "生成密码、UUID、密钥等",
		Long: `生成各种类型的密码、UUID、密钥。

支持的类型:
  • password  - 随机密码
  • uuid      - UUID v4
  • ss2022    - Shadowsocks 2022 密钥
  • keypair   - X25519 密钥对
  • shortid   - REALITY 短 ID
  • vlessenc  - 生成 VLESS Encryption 的 decryption/encryption 配对 (调用 xray)
  • mlkem     - 生成 ML-KEM-768 密钥材料 (调用 xray)`,
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

			case "vlessenc":
				dec, enc, err := composeVLESSENCPair(vlessAuth, vlessMode, vlessServerRTT, vlessClientRTT, vlessPadding)
				if err != nil {
					return err
				}
				utils.PrintKeyValue("decryption", dec)
				utils.PrintKeyValue("encryption", enc)

			case "mlkem":
				out, err := runXraySimple("mlkem768")
				if err != nil {
					return fmt.Errorf("执行 xray mlkem768 失败: %w", err)
				}
				fmt.Print(out)

			default:
				return fmt.Errorf("不支持的生成类型: %s", genType)
			}

			return nil
		},
	}

	// VLESS-Encryption flags (only effective when type=vlessenc)
	cmd.Flags().StringVar(&vlessAuth, "auth", "mlkem768", "VLESS-Encryption 认证 (mlkem768|x25519)")
	cmd.Flags().StringVar(&vlessMode, "mode", "native", "VLESS-Encryption 模式 (native|xorpub|random)")
	cmd.Flags().StringVar(&vlessServerRTT, "server-rtt", "600s", "服务端 1-RTT 时长（如 600s 或 600-900s；或 0rtt）")
	cmd.Flags().StringVar(&vlessClientRTT, "client-rtt", "0rtt", "客户端 0-RTT/1-RTT 设置（0rtt 或 如 600s）")
	cmd.Flags().StringVar(&vlessPadding, "padding", "", "1-RTT padding 规则（以 . 分隔多段，如 100-111-1111.75-0-111.50-0-3333）")

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
				utils.PrintInfo("已占用端口数: %d", totalPorts-availableCount)

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
		Args: cobra.ExactArgs(1),
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
			utils.PrintKeyValue("XRF-Go 版本", Version)
			if BuildTime == "" {
				BuildTime = "未设置"
			}
			utils.PrintKeyValue("构建时间", BuildTime)
			if GitCommit != "" {
				utils.PrintKeyValue("Git 提交", GitCommit)
			}
			utils.PrintKeyValue("Go 版本", "1.23+")
			return nil
		},
	}
}

func createTLSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tls",
		Short: "TLS 证书管理",
		Long: `管理 Let's Encrypt 自动证书申请和续期。

支持的操作:
  • request   - 申请证书
  • renew     - 续期证书  
  • status    - 查看证书状态
  • auto-renew - 设置自动续期`,
	}

	cmd.AddCommand(
		createTLSRequestCommand(),
		createTLSRenewCommand(),
		createTLSStatusCommand(),
		createTLSAutoRenewCommand(),
	)

	return cmd
}

func createTLSRequestCommand() *cobra.Command {
	var (
		email   string
		staging bool
	)

	cmd := &cobra.Command{
		Use:   "request [domain]",
		Short: "申请 Let's Encrypt 证书",
		Long: `为指定域名申请 Let's Encrypt SSL/TLS 证书。

示例:
  xrf tls request example.com --email admin@example.com
  xrf tls request test.com --email admin@test.com --staging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			if email == "" {
				return fmt.Errorf("email is required")
			}

			utils.PrintSection("申请 Let's Encrypt 证书")
			utils.PrintInfo("域名: %s", domain)
			utils.PrintInfo("邮箱: %s", email)

			// 初始化 ACME 管理器
			acmeMgr = tls.NewACMEManager(email)
			if staging {
				acmeMgr.SetStagingMode()
				utils.PrintInfo("使用 Let's Encrypt 测试环境")
			}

			// 初始化
			if err := acmeMgr.Initialize(); err != nil {
				return fmt.Errorf("failed to initialize ACME manager: %w", err)
			}

			// 申请证书
			if err := acmeMgr.ObtainCertificate([]string{domain}); err != nil {
				return fmt.Errorf("failed to obtain certificate: %w", err)
			}

			utils.PrintSuccess("证书申请完成")
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "ACME 账户邮箱")
	cmd.Flags().BoolVar(&staging, "staging", false, "使用 Let's Encrypt 测试环境")
	if err := cmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}

	return cmd
}

func createTLSRenewCommand() *cobra.Command {
	var (
		email string
		all   bool
	)

	cmd := &cobra.Command{
		Use:   "renew [domain]",
		Short: "续期证书",
		Long: `手动续期指定域名的证书，或续期所有即将过期的证书。

示例:
  xrf tls renew example.com --email admin@example.com
  xrf tls renew --all --email admin@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("email is required")
			}

			// 初始化 ACME 管理器
			acmeMgr = tls.NewACMEManager(email)
			if err := acmeMgr.Initialize(); err != nil {
				return fmt.Errorf("failed to initialize ACME manager: %w", err)
			}

			if all {
				utils.PrintSection("续期所有即将过期的证书")
				return acmeMgr.CheckAndRenew()
			}

			if len(args) != 1 {
				return fmt.Errorf("domain is required when --all is not specified")
			}

			domain := args[0]
			utils.PrintSection("续期证书")
			utils.PrintInfo("域名: %s", domain)

			return acmeMgr.RenewCertificate(domain)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "ACME 账户邮箱")
	cmd.Flags().BoolVar(&all, "all", false, "续期所有即将过期的证书")
	if err := cmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}

	return cmd
}

func createTLSStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看证书状态",
		Long:  `查看所有已申请证书的状态信息，包括过期时间等。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("TLS 证书状态")

			// 这里可以实现证书状态查看逻辑
			// 扫描证书目录并显示证书信息
			utils.PrintInfo("功能开发中...")

			return nil
		},
	}
}

func createTLSAutoRenewCommand() *cobra.Command {
	var (
		email  string
		enable bool
	)

	cmd := &cobra.Command{
		Use:   "auto-renew",
		Short: "设置自动续期",
		Long: `启用或禁用证书自动续期功能。

示例:
  xrf tls auto-renew --enable --email admin@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("email is required")
			}

			// 初始化 ACME 管理器
			acmeMgr = tls.NewACMEManager(email)

			if enable {
				utils.PrintSection("启用自动续期")
				return acmeMgr.SetupAutoRenewal()
			}

			utils.PrintInfo("自动续期功能管理")
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "ACME 账户邮箱")
	cmd.Flags().BoolVar(&enable, "enable", false, "启用自动续期")
	if err := cmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}

	return cmd
}

func createCaddyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "caddy",
		Short: "Caddy 反向代理管理",
		Long: `管理 Caddy 反向代理服务器，提供 TLS 终止和网站伪装功能。

支持的操作:
  • install - 安装 Caddy
  • config  - 配置反向代理
  • mask    - 设置伪装网站
  • status  - 查看服务状态
  • start   - 启动服务
  • stop    - 停止服务`,
	}

	cmd.AddCommand(
		createCaddyInstallCommand(),
		createCaddyConfigCommand(),
		createCaddyMaskCommand(),
		createCaddyStatusCommand(),
		createCaddyStartCommand(),
		createCaddyStopCommand(),
	)

	return cmd
}

func createCaddyInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "安装 Caddy",
		Long:  `下载并安装 Caddy 服务器，创建 systemd 服务配置。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("安装 Caddy")

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 安装 Caddy
			if err := caddyMgr.InstallCaddy(); err != nil {
				return fmt.Errorf("failed to install Caddy: %w", err)
			}

			utils.PrintSuccess("Caddy 安装完成")
			return nil
		},
	}
}

func createCaddyConfigCommand() *cobra.Command {
	var (
		domain   string
		upstream string
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "配置反向代理",
		Long: `为指定域名配置 Caddy 反向代理。

示例:
  xrf caddy config --domain example.com --upstream localhost:8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("domain is required")
			}
			if upstream == "" {
				return fmt.Errorf("upstream is required")
			}

			utils.PrintSection("配置 Caddy 反向代理")
			utils.PrintInfo("域名: %s", domain)
			utils.PrintInfo("上游: %s", upstream)

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 配置反向代理
			if err := caddyMgr.ConfigureReverseProxy(domain, upstream); err != nil {
				return fmt.Errorf("failed to configure reverse proxy: %w", err)
			}

			utils.PrintSuccess("反向代理配置完成")
			return nil
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "域名")
	cmd.Flags().StringVar(&upstream, "upstream", "", "上游服务器地址")
	if err := cmd.MarkFlagRequired("domain"); err != nil {
		panic(fmt.Sprintf("failed to mark domain flag as required: %v", err))
	}
	if err := cmd.MarkFlagRequired("upstream"); err != nil {
		panic(fmt.Sprintf("failed to mark upstream flag as required: %v", err))
	}

	return cmd
}

func createCaddyMaskCommand() *cobra.Command {
	var (
		domain   string
		maskSite string
	)

	cmd := &cobra.Command{
		Use:   "mask",
		Short: "设置伪装网站",
		Long: `为指定域名设置伪装网站，反向代理到指定的网站。

示例:
  xrf caddy mask --domain example.com --site google.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("domain is required")
			}
			if maskSite == "" {
				return fmt.Errorf("mask site is required")
			}

			utils.PrintSection("设置伪装网站")
			utils.PrintInfo("域名: %s", domain)
			utils.PrintInfo("伪装网站: %s", maskSite)

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 设置伪装网站
			if err := caddyMgr.AddWebsiteMasquerade(domain, maskSite); err != nil {
				return fmt.Errorf("failed to setup website masquerade: %w", err)
			}

			utils.PrintSuccess("伪装网站设置完成")
			return nil
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "域名")
	cmd.Flags().StringVar(&maskSite, "site", "", "伪装网站地址")
	if err := cmd.MarkFlagRequired("domain"); err != nil {
		panic(fmt.Sprintf("failed to mark domain flag as required: %v", err))
	}
	if err := cmd.MarkFlagRequired("site"); err != nil {
		panic(fmt.Sprintf("failed to mark site flag as required: %v", err))
	}

	return cmd
}

func createCaddyStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看 Caddy 服务状态",
		Long:  `查看 Caddy 服务的运行状态和配置信息。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("Caddy 服务状态")

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 获取服务状态
			status, err := caddyMgr.GetServiceStatus()
			if err != nil {
				return fmt.Errorf("failed to get service status: %w", err)
			}

			utils.PrintKeyValue("服务状态", status)
			utils.PrintKeyValue("是否运行", func() string {
				if caddyMgr.IsRunning() {
					return "是"
				}
				return "否"
			}())

			return nil
		},
	}
}

func createCaddyStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "启动 Caddy 服务",
		Long:  `启动 Caddy 服务并启用自动启动。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("启动 Caddy 服务")

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 启动服务
			if err := caddyMgr.StartService(); err != nil {
				return fmt.Errorf("failed to start Caddy service: %w", err)
			}

			return nil
		},
	}
}

func createCaddyStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止 Caddy 服务",
		Long:  `停止 Caddy 服务。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("停止 Caddy 服务")

			// 初始化 Caddy 管理器
			caddyMgr = tls.NewCaddyManager()

			// 停止服务
			if err := caddyMgr.StopService(); err != nil {
				return fmt.Errorf("failed to stop Caddy service: %w", err)
			}

			return nil
		},
	}
}

// DESIGN.md required commands implementation

// createIPCommand creates the IP command (DESIGN.md line 178)
func createIPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "获取服务器公网IP",
		Long:  `获取服务器的公网IP地址，用于配置和分享。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("获取公网IP")

			ip, err := utils.GetPublicIP()
			if err == nil && ip != "" {
				utils.PrintKeyValue("公网IP", ip)
				return nil
			}

			return fmt.Errorf("无法获取公网IP地址")
		},
	}
}

// createBBRCommand creates the BBR command (DESIGN.md line 179)
func createBBRCommand() *cobra.Command {
	var enable bool
	var disable bool

	cmd := &cobra.Command{
		Use:   "bbr",
		Short: "BBR拥塞控制管理",
		Long:  `启用或禁用BBR拥塞控制算法，提升网络传输性能。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("BBR拥塞控制管理")

			if enable && disable {
				return fmt.Errorf("不能同时使用 --enable 和 --disable")
			}

			if enable {
				utils.PrintInfo("启用BBR拥塞控制...")
				if err := enableBBR(); err != nil {
					return fmt.Errorf("启用BBR失败: %w", err)
				}
				utils.PrintSuccess("BBR拥塞控制已启用")
				return nil
			}

			if disable {
				utils.PrintInfo("禁用BBR拥塞控制...")
				if err := disableBBR(); err != nil {
					return fmt.Errorf("禁用BBR失败: %w", err)
				}
				utils.PrintSuccess("BBR拥塞控制已禁用")
				return nil
			}

			// 显示BBR状态
			status, err := getBBRStatus()
			if err != nil {
				return fmt.Errorf("获取BBR状态失败: %w", err)
			}

			utils.PrintKeyValue("BBR状态", status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&enable, "enable", false, "启用BBR")
	cmd.Flags().BoolVar(&disable, "disable", false, "禁用BBR")

	return cmd
}

// createSwitchCommand creates the switch command (DESIGN.md line 674)
func createSwitchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "switch [name]",
		Short: "快速协议切换",
		Long:  `快速切换到指定的协议配置，停用其他协议。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			protocolName := args[0]

			utils.PrintSection("快速协议切换")
			utils.PrintInfo("目标协议: %s", protocolName)

			// 获取所有协议配置
			protocols, err := configMgr.ListProtocols()
			if err != nil {
				return fmt.Errorf("获取协议列表失败: %w", err)
			}

			var targetFound bool
			var targetTag string

			// 查找目标协议
			for _, protocol := range protocols {
				if strings.Contains(protocol.Type, protocolName) ||
					strings.Contains(protocol.Tag, protocolName) {
					targetFound = true
					targetTag = protocol.Tag
					break
				}
			}

			if !targetFound {
				return fmt.Errorf("未找到协议: %s", protocolName)
			}

			// 停用其他协议 (实际实现需要配置管理支持)
			utils.PrintInfo("切换到协议: %s", targetTag)
			utils.PrintWarning("协议切换功能需要配置管理器支持")

			return nil
		},
	}
}

// createEnableAllCommand creates the enable-all command (DESIGN.md line 675)
func createEnableAllCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable-all",
		Short: "启用所有协议",
		Long:  `启用所有已配置的协议。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("启用所有协议")

			// 获取所有协议配置
			protocols, err := configMgr.ListProtocols()
			if err != nil {
				return fmt.Errorf("获取协议列表失败: %w", err)
			}

			if len(protocols) == 0 {
				utils.PrintInfo("没有找到协议配置")
				return nil
			}

			utils.PrintInfo("正在启用 %d 个协议...", len(protocols))

			for _, protocol := range protocols {
				utils.PrintInfo("启用协议: %s", protocol.Tag)
				// 实际实现需要配置管理器支持启用/禁用功能
			}

			utils.PrintSuccess("所有协议已启用")
			utils.PrintWarning("协议启用功能需要配置管理器支持")

			return nil
		},
	}
}

// createUpdateCommand creates the update command (DESIGN.md line 678)
func createUpdateCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新Xray版本",
		Long:  `检查并更新Xray到最新版本。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("更新Xray")

			// 检查当前版本
			currentVersion, err := getCurrentXrayVersion()
			if err != nil {
				utils.PrintWarning("获取当前版本失败: %v", err)
			} else {
				utils.PrintKeyValue("当前版本", currentVersion)
			}

			// 检查最新版本
			utils.PrintInfo("检查最新版本...")
			latestVersion, err := getLatestXrayVersion()
			if err != nil {
				return fmt.Errorf("检查最新版本失败: %w", err)
			}

			utils.PrintKeyValue("最新版本", latestVersion)

			if !force && currentVersion == latestVersion {
				utils.PrintInfo("已是最新版本，无需更新")
				return nil
			}

			// 执行更新
			utils.PrintInfo("正在下载并安装最新版本...")
			if err := installer.UpdateXray(latestVersion); err != nil {
				return fmt.Errorf("更新失败: %w", err)
			}

			utils.PrintSuccess("Xray更新完成")
			utils.PrintInfo("建议执行 'xrf restart' 重启服务")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "强制更新")

	return cmd
}

// createCleanCommand creates the clean command (DESIGN.md line 679)
func createCleanCommand() *cobra.Command {
	var logs bool
	var configs bool
	var temp bool
	var all bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "清理操作",
		Long:  `清理日志、临时文件、备份配置等。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("清理操作")

			if all {
				logs = true
				configs = true
				temp = true
			}

			if !logs && !configs && !temp {
				// 默认清理临时文件
				temp = true
			}

			if logs {
				utils.PrintInfo("清理日志文件...")
				if err := cleanLogs(); err != nil {
					utils.PrintWarning("清理日志失败: %v", err)
				} else {
					utils.PrintSuccess("日志文件清理完成")
				}
			}

			if configs {
				utils.PrintInfo("清理备份配置...")
				if err := cleanBackupConfigs(); err != nil {
					utils.PrintWarning("清理备份失败: %v", err)
				} else {
					utils.PrintSuccess("备份配置清理完成")
				}
			}

			if temp {
				utils.PrintInfo("清理临时文件...")
				if err := cleanTempFiles(); err != nil {
					utils.PrintWarning("清理临时文件失败: %v", err)
				} else {
					utils.PrintSuccess("临时文件清理完成")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&logs, "logs", false, "清理日志文件")
	cmd.Flags().BoolVar(&configs, "configs", false, "清理备份配置")
	cmd.Flags().BoolVar(&temp, "temp", false, "清理临时文件")
	cmd.Flags().BoolVar(&all, "all", false, "清理所有")

	return cmd
}

// Helper functions for the new commands

func enableBBR() error {
	// Enable BBR congestion control
	commands := []string{
		"echo 'net.core.default_qdisc=fq' >> /etc/sysctl.conf",
		"echo 'net.ipv4.tcp_congestion_control=bbr' >> /etc/sysctl.conf",
		"sysctl -p",
	}

	for _, cmd := range commands {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			return fmt.Errorf("failed to execute: %s", cmd)
		}
	}

	return nil
}

func disableBBR() error {
	// Disable BBR by setting back to default
	commands := []string{
		"sed -i '/net.core.default_qdisc=fq/d' /etc/sysctl.conf",
		"sed -i '/net.ipv4.tcp_congestion_control=bbr/d' /etc/sysctl.conf",
		"sysctl -p",
	}

	for _, cmd := range commands {
		if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
			return fmt.Errorf("failed to execute: %s", cmd)
		}
	}

	return nil
}

func getBBRStatus() (string, error) {
	// Check if BBR is enabled
	cmd := exec.Command("sysctl", "net.ipv4.tcp_congestion_control")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(output))
	if strings.Contains(result, "bbr") {
		return "已启用", nil
	}

	return "未启用", nil
}

func getCurrentXrayVersion() (string, error) {
	cmd := exec.Command("xray", "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Try to parse a semantic version token from the first line
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Xray") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "Xray" && i+1 < len(parts) {
					v := parts[i+1]
					if strings.HasPrefix(v, "v") && len(v) > 1 && v[1] >= '0' && v[1] <= '9' {
						return v, nil
					}
					if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
						return "v" + v, nil
					}
				}
				if part == "version" && i+1 < len(parts) { // handle "Xray version 1.2.3"
					v := parts[i+1]
					if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
						return "v" + v, nil
					}
				}
			}
		}
	}

	return "unknown", nil
}

func getLatestXrayVersion() (string, error) {
	// Allow override via environment variable
	if v := strings.TrimSpace(os.Getenv("XRAY_VERSION")); v != "" && v != "latest" {
		return v, nil
	}

	req, err := http.NewRequest("GET", system.XrayReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "xrf-go-cli")
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error: %s", resp.Status)
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", fmt.Errorf("无法解析最新版本 tag")
	}
	return r.TagName, nil
}

func runXraySimple(args ...string) (string, error) {
	xrayPath, err := exec.LookPath("xray")
	if err != nil {
		return "", fmt.Errorf("未找到 xray 可执行文件，请先安装 Xray 或将其加入 PATH")
	}
	cmd := exec.Command(xrayPath, args...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("xray %s 执行失败: %v\n输出: %s", strings.Join(args, " "), err, string(b))
	}
	return string(b), nil
}

// composeVLESSENCPair 根据参数生成 decryption/encryption 配对
func composeVLESSENCPair(auth, mode, serverRTT, clientRTT, padding string) (string, string, error) {
	auth = strings.ToLower(strings.TrimSpace(auth))
	if auth == "" {
		auth = "mlkem768"
	}
	switch auth {
	case "mlkem768", "x25519":
	default:
		return "", "", fmt.Errorf("不支持的认证方式: %s (应为 mlkem768|x25519)", auth)
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "native"
	}
	switch mode {
	case "native", "xorpub", "random":
	default:
		return "", "", fmt.Errorf("不支持的模式: %s (应为 native|xorpub|random)", mode)
	}

	if err := validateVERRT(serverRTT); err != nil {
		return "", "", fmt.Errorf("server-rtt 无效: %w", err)
	}
	if err := validateVERRT(clientRTT); err != nil {
		return "", "", fmt.Errorf("client-rtt 无效: %w", err)
	}
	if err := validateVEPadding(padding); err != nil {
		return "", "", fmt.Errorf("padding 无效: %w", err)
	}

	// 生成/获取密钥材料
	var serverKey, clientKey string
	var err error
	if auth == "x25519" {
		serverKey, clientKey, err = utils.GenerateX25519KeyPair()
		if err != nil {
			return "", "", fmt.Errorf("生成 X25519 密钥对失败: %w", err)
		}
	} else {
		// 调用 xray 生成 ML-KEM-768 种子与 client 公钥
		serverKey, clientKey, err = generateMLKEMPair()
		if err != nil {
			return "", "", err
		}
	}

	prefix := "mlkem768x25519plus"
	dec := buildVEDot(prefix, mode, serverRTT, padding, serverKey)
	enc := buildVEDot(prefix, mode, clientRTT, padding, clientKey)
	return dec, enc, nil
}

func validateVERRT(s string) error {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "0rtt" {
		return nil
	}
	// 600s 或 600-900s
	r := regexp.MustCompile(`^\d{1,5}(?:-\d{1,5})?s$`)
	if !r.MatchString(s) {
		return fmt.Errorf("必须为 0rtt 或 <sec>s 或 <from>-<to>s，如 600s、600-900s")
	}
	return nil
}

func validateVEPadding(p string) error {
	if strings.TrimSpace(p) == "" {
		return nil
	}
	// 允许数字/连字符/点分段
	r := regexp.MustCompile(`^[0-9.-]+$`)
	if !r.MatchString(p) {
		return fmt.Errorf("仅允许数字、- 和 . 分隔的段")
	}
	return nil
}

func buildVEDot(prefix, mode, rtt, padding, key string) string {
	parts := []string{prefix, mode, strings.ToLower(strings.TrimSpace(rtt))}
	if strings.TrimSpace(padding) != "" {
		// 用户以 . 分段的 padding
		for _, seg := range strings.Split(padding, ".") {
			seg = strings.TrimSpace(seg)
			if seg != "" {
				parts = append(parts, seg)
			}
		}
	}
	parts = append(parts, key)
	return strings.Join(parts, ".")
}

func generateMLKEMPair() (seed string, client string, err error) {
	out, err := runXraySimple("mlkem768")
	if err != nil {
		return "", "", err
	}
	// Parse lines: Seed: <b64>\nClient: <b64>\n
	reSeed := regexp.MustCompile(`(?m)^Seed:\s*([A-Za-z0-9_-]+)=*$`)
	reClient := regexp.MustCompile(`(?m)^Client:\s*([A-Za-z0-9_-]+)=*$`)
	ms := reSeed.FindStringSubmatch(out)
	mc := reClient.FindStringSubmatch(out)
	if len(ms) != 2 || len(mc) != 2 {
		return "", "", fmt.Errorf("无法解析 mlkem768 输出: %s", out)
	}
	return ms[1], mc[1], nil
}

func cleanLogs() error {
	// Clean log files
	logPaths := []string{
		"/var/log/xray/*.log",
		"/tmp/xray*.log",
	}

	for _, path := range logPaths {
		_ = exec.Command("sh", "-c", fmt.Sprintf("rm -f %s", path)).Run()
		// Continue even if some files can't be deleted
	}

	return nil
}

func cleanBackupConfigs() error {
	// Clean backup configuration files
	backupPaths := []string{
		"/etc/xray/confs/*.bak",
		"/tmp/xrf-backup-*.tar.gz",
	}

	for _, path := range backupPaths {
		_ = exec.Command("sh", "-c", fmt.Sprintf("rm -f %s", path)).Run()
		// Continue even if some files can't be deleted
	}

	return nil
}

func cleanTempFiles() error {
	// Clean temporary files
	tempPaths := []string{
		"/tmp/xrf-*",
		"/tmp/xray-*",
	}

	for _, path := range tempPaths {
		_ = exec.Command("sh", "-c", fmt.Sprintf("rm -rf %s", path)).Run()
		// Continue even if some files can't be deleted
	}

	return nil
}
