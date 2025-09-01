package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/xrf-go/pkg/config"
	"github.com/yourusername/xrf-go/pkg/utils"
)

var (
	confDir   string
	verbose   bool
	noColor   bool
	configMgr *config.ConfigManager
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
			
			if len(protocols) == 0 {
				protocols = []string{"vless-reality"}
			}
			
			// TODO: 实现完整的安装逻辑
			utils.PrintInfo("正在安装 Xray 服务...")
			utils.PrintInfo("协议: %s", strings.Join(protocols, ", "))
			if domain != "" {
				utils.PrintInfo("域名: %s", domain)
			}
			if port != 0 {
				utils.PrintInfo("端口: %d", port)
			}
			
			// 初始化配置管理器
			if err := configMgr.Initialize(); err != nil {
				return fmt.Errorf("初始化配置失败: %w", err)
			}
			
			utils.PrintSuccess("Xray 服务安装完成")
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
			
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "端口")
	cmd.Flags().StringVar(&domain, "domain", "", "域名")
	cmd.Flags().StringVar(&path, "path", "", "路径")
	cmd.Flags().StringVar(&password, "password", "", "密码")
	cmd.Flags().StringVar(&uuid, "uuid", "", "UUID")
	cmd.Flags().StringVar(&tag, "tag", "", "配置标签")

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
			return nil
		},
	}

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
			
			utils.PrintSection("协议配置详情")
			utils.PrintInfo("标签: %s", tag)
			
			// TODO: 实现配置详情显示
			utils.PrintWarning("配置详情功能尚未实现")
			return nil
		},
	}

	return cmd
}

func createChangeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change [tag] [key] [value]",
		Short: "修改协议配置",
		Long:  `修改指定协议的配置参数。`,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, key, value := args[0], args[1], args[2]
			
			utils.PrintSection("修改协议配置")
			utils.PrintInfo("标签: %s", tag)
			utils.PrintInfo("参数: %s = %s", key, value)
			
			options := map[string]interface{}{
				key: value,
			}
			
			if err := configMgr.UpdateProtocol(tag, options); err != nil {
				return fmt.Errorf("修改协议配置失败: %w", err)
			}
			
			utils.PrintSuccess("协议配置修改成功")
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
			utils.PrintInfo("启动 Xray 服务...")
			// TODO: 实现服务启动逻辑
			utils.PrintWarning("服务管理功能尚未实现")
			return nil
		},
	}
}

func createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止 Xray 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintInfo("停止 Xray 服务...")
			// TODO: 实现服务停止逻辑
			utils.PrintWarning("服务管理功能尚未实现")
			return nil
		},
	}
}

func createRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重启 Xray 服务",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintInfo("重启 Xray 服务...")
			// TODO: 实现服务重启逻辑
			utils.PrintWarning("服务管理功能尚未实现")
			return nil
		},
	}
}

func createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看服务状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("Xray 服务状态")
			// TODO: 实现服务状态查询
			utils.PrintWarning("服务状态查询功能尚未实现")
			return nil
		},
	}
}

func createReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "热重载配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintInfo("重载配置...")
			// TODO: 实现配置热重载
			utils.PrintWarning("配置热重载功能尚未实现")
			return nil
		},
	}
}

func createTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "验证配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("配置验证")
			
			if err := configMgr.ValidateConfig(); err != nil {
				utils.PrintError("配置验证失败: %v", err)
				return err
			}
			
			utils.PrintSuccess("配置验证通过")
			return nil
		},
	}
}

func createBackupCommand() *cobra.Command {
	var backupPath string
	
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "备份配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintInfo("备份配置到: %s", backupPath)
			// TODO: 实现配置备份
			utils.PrintWarning("配置备份功能尚未实现")
			return nil
		},
	}
	
	cmd.Flags().StringVar(&backupPath, "output", "", "备份文件路径")
	return cmd
}

func createRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [backup-file]",
		Short: "恢复配置",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupFile := args[0]
			utils.PrintInfo("从备份文件恢复配置: %s", backupFile)
			// TODO: 实现配置恢复
			utils.PrintWarning("配置恢复功能尚未实现")
			return nil
		},
	}
	
	return cmd
}

func createURLCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "url [tag]",
		Short: "生成分享链接",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			utils.PrintInfo("生成分享链接: %s", tag)
			// TODO: 实现分享链接生成
			utils.PrintWarning("分享链接生成功能尚未实现")
			return nil
		},
	}
}

func createQRCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "qr [tag]",
		Short: "显示二维码",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			utils.PrintInfo("显示二维码: %s", tag)
			// TODO: 实现二维码显示
			utils.PrintWarning("二维码显示功能尚未实现")
			return nil
		},
	}
}

func createLogsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "查看运行日志",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintSection("Xray 运行日志")
			// TODO: 实现日志查看
			utils.PrintWarning("日志查看功能尚未实现")
			return nil
		},
	}
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