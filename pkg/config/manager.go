package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Joe-oss9527/xrf-go/pkg/utils"
)

const (
	DefaultConfDir = "/etc/xray/confs"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	confDir   string
	renderer  *TemplateRenderer
	protocols *ProtocolManager
}

// ConfigFile 配置文件信息
type ConfigFile struct {
	Priority int    `json:"priority"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
	IsTail   bool   `json:"is_tail"`
}

// NewConfigManager 创建配置管理器
func NewConfigManager(confDir string) *ConfigManager {
	if confDir == "" {
		confDir = DefaultConfDir
	}

	return &ConfigManager{
		confDir:   confDir,
		renderer:  NewTemplateRenderer(),
		protocols: NewProtocolManager(),
	}
}

// Initialize 初始化配置目录和基础配置
func (cm *ConfigManager) Initialize() error {
	// 创建配置目录
	if err := os.MkdirAll(cm.confDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 创建基础配置文件
	if err := cm.createBaseConfigs(); err != nil {
		return fmt.Errorf("failed to create base configs: %w", err)
	}

	utils.Info("Initialized XRF configuration directory: %s", cm.confDir)
	return nil
}

// createBaseConfigs 创建基础配置文件
func (cm *ConfigManager) createBaseConfigs() error {
	baseConfigs := []struct {
		filename string
		template string
		data     TemplateData
	}{
		{
			filename: "00-base.json",
			template: BaseConfigTemplate,
			data:     TemplateData{},
		},
		{
			filename: "01-dns.json",
			template: DNSConfigTemplate,
			data:     TemplateData{},
		},
		{
			filename: "20-outbound-direct.json",
			template: DirectOutboundTemplate,
			data:     TemplateData{},
		},
		{
			filename: "21-outbound-block.json",
			template: BlockOutboundTemplate,
			data:     TemplateData{},
		},
		{
			filename: "90-routing-basic.json",
			template: BasicRoutingTemplate,
			data:     TemplateData{},
		},
		{
			filename: "99-routing-tail.json",
			template: TailRoutingTemplate,
			data:     TemplateData{},
		},
	}

	for _, config := range baseConfigs {
		configPath := filepath.Join(cm.confDir, config.filename)

		// 如果文件已存在，跳过
		if _, err := os.Stat(configPath); err == nil {
			continue
		}

		content, err := cm.renderer.Render(config.template, config.data)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", config.filename, err)
		}

		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", config.filename, err)
		}

		utils.Debug("Created base config: %s", config.filename)
	}

	return nil
}

// AddProtocol 添加协议配置 (Enhanced with DESIGN.md rollback requirement)
func (cm *ConfigManager) AddProtocol(protocolType, tag string, options map[string]interface{}) error {
	// DESIGN.md requirement (line 575): 操作失败自动回滚
	// Create automatic backup before changes
	backupPath, err := cm.createAutoBackup("add-protocol-" + tag)
	if err != nil {
		utils.Warning("Failed to create backup before adding protocol: %v", err)
	}

	// Rollback function in case of failure
	rollback := func() {
		if backupPath != "" {
			if rollbackErr := cm.restoreFromBackup(backupPath); rollbackErr != nil {
				utils.Error("Failed to rollback after error: %v", rollbackErr)
			} else {
				utils.Info("Configuration rolled back due to failure")
			}
		}
	}

	// 检查标签是否已存在
	_, err = cm.GetProtocolInfo(tag)
	if err == nil {
		rollback()
		return &utils.XRFError{
			Type:    utils.ErrConfigConflict,
			Message: fmt.Sprintf("协议标签 '%s' 已存在", tag),
			Context: map[string]interface{}{
				"tag":      tag,
				"protocol": protocolType,
			},
		}
	}

	protocol, err := cm.protocols.GetProtocol(protocolType)
	if err != nil {
		rollback()
		return utils.NewProtocolNotSupportedError(protocolType)
	}

	// 生成模板数据
	data, err := cm.generateTemplateData(protocol, tag, options)
	if err != nil {
		rollback()
		return fmt.Errorf("failed to generate template data: %w", err)
	}

	// 获取协议模板
	templateStr, exists := GetProtocolTemplate(protocol.Template)
	if !exists {
		rollback()
		return fmt.Errorf("template not found for protocol: %s", protocol.Template)
	}

	// 渲染配置模板
	content, err := cm.renderer.Render(templateStr, data)
	if err != nil {
		rollback()
		return fmt.Errorf("failed to render protocol template: %w", err)
	}

	// 生成配置文件名
	filename := cm.generateConfigFilename("inbound", tag, false)
	configPath := filepath.Join(cm.confDir, filename)

	// 写入配置文件
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		rollback()
		return fmt.Errorf("failed to write protocol config: %w", err)
	}

	// 验证配置是否有效（可通过环境变量跳过）
	if os.Getenv("XRF_SKIP_VALIDATION") != "1" {
		if err := cm.validateConfigAfterChange(); err != nil {
			rollback()
			return fmt.Errorf("configuration validation failed after adding protocol: %w", err)
		}
	} else {
		utils.Debug("Skipping validation due to XRF_SKIP_VALIDATION=1")
	}

	// Success - cleanup backup
	if backupPath != "" {
		os.Remove(backupPath)
	}

	utils.Success("Added protocol %s with tag %s", protocol.Name, tag)
	return nil
}

// RemoveProtocol 删除协议配置 (Enhanced with DESIGN.md rollback requirement)
func (cm *ConfigManager) RemoveProtocol(tag string) error {
	// DESIGN.md requirement (line 575): 操作失败自动回滚
	// Create automatic backup before changes
	backupPath, err := cm.createAutoBackup("remove-protocol-" + tag)
	if err != nil {
		utils.Warning("Failed to create backup before removing protocol: %v", err)
	}

	// Rollback function in case of failure
	rollback := func() {
		if backupPath != "" {
			if rollbackErr := cm.restoreFromBackup(backupPath); rollbackErr != nil {
				utils.Error("Failed to rollback after error: %v", rollbackErr)
			} else {
				utils.Info("Configuration rolled back due to failure")
			}
		}
	}

	// 查找包含该 tag 的配置文件
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		rollback()
		return err
	}

	if len(files) == 0 {
		rollback()
		return fmt.Errorf("protocol with tag '%s' not found", tag)
	}

	// 删除配置文件
	var failedFiles []string
	for _, file := range files {
		if err := os.Remove(file.Path); err != nil {
			failedFiles = append(failedFiles, file.Filename)
			utils.Warning("Failed to remove config file %s: %v", file.Filename, err)
		} else {
			utils.Debug("Removed config file: %s", file.Filename)
		}
	}

	// Check if any files failed to remove
	if len(failedFiles) > 0 {
		rollback()
		return fmt.Errorf("failed to remove configuration files: %v", failedFiles)
	}

	// 验证配置是否有效（可通过环境变量跳过）
	if os.Getenv("XRF_SKIP_VALIDATION") != "1" {
		if err := cm.validateConfigAfterChange(); err != nil {
			rollback()
			return fmt.Errorf("configuration validation failed after removing protocol: %w", err)
		}
	}

	// Success - cleanup backup
	if backupPath != "" {
		os.Remove(backupPath)
	}

	utils.Success("Removed protocol with tag %s", tag)
	return nil
}

// ListProtocols 列出所有协议配置
func (cm *ConfigManager) ListProtocols() ([]ProtocolInfo, error) {
	var protocols []ProtocolInfo

	files, err := cm.listConfigFiles()
	if err != nil {
		return nil, err
	}

	// 过滤入站配置文件
	for _, file := range files {
		if file.Type == "inbound" {
			info, err := cm.parseProtocolInfo(file)
			if err != nil {
				utils.Warning("Failed to parse protocol info from %s: %v", file.Filename, err)
				continue
			}
			protocols = append(protocols, info)
		}
	}

	return protocols, nil
}

// UpdateProtocol 更新协议配置 (Enhanced with DESIGN.md rollback requirement)
func (cm *ConfigManager) UpdateProtocol(tag string, options map[string]interface{}) error {
	// DESIGN.md requirement (line 575): 操作失败自动回滚
	// Create automatic backup before changes
	backupPath, err := cm.createAutoBackup("update-protocol-" + tag)
	if err != nil {
		utils.Warning("Failed to create backup before updating protocol: %v", err)
	}

	// Rollback function in case of failure
	rollback := func() {
		if backupPath != "" {
			if rollbackErr := cm.restoreFromBackup(backupPath); rollbackErr != nil {
				utils.Error("Failed to rollback after error: %v", rollbackErr)
			} else {
				utils.Info("Configuration rolled back due to failure")
			}
		}
	}

	// 查找现有配置
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		rollback()
		return err
	}

	if len(files) == 0 {
		rollback()
		return fmt.Errorf("protocol with tag '%s' not found", tag)
	}

	// 读取现有配置
	configFile := files[0]
	existingConfig, err := cm.readConfigFile(configFile.Path)
	if err != nil {
		rollback()
		return err
	}

	// 更新配置
	if err := cm.mergeConfigOptions(existingConfig, options); err != nil {
		rollback()
		return err
	}

	// 写回配置文件
	updatedContent, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		rollback()
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(configFile.Path, updatedContent, 0644); err != nil {
		rollback()
		return fmt.Errorf("failed to write updated config: %w", err)
	}

	// 验证配置是否有效（可通过环境变量跳过）
	if os.Getenv("XRF_SKIP_VALIDATION") != "1" {
		if err := cm.validateConfigAfterChange(); err != nil {
			rollback()
			return fmt.Errorf("configuration validation failed after updating protocol: %w", err)
		}
	}

	// Success - cleanup backup
	if backupPath != "" {
		os.Remove(backupPath)
	}

	utils.Success("Updated protocol %s", tag)
	return nil
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig() error {
	return utils.ValidateXrayConfDir(cm.confDir)
}

// ReloadConfig 热重载配置（发送USR1信号到Xray进程）
func (cm *ConfigManager) ReloadConfig() error {
	utils.PrintInfo("重载 Xray 配置...")

	// 首先验证配置
	if err := cm.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败，取消重载: %w", err)
	}

	// 查找 Xray 进程
	cmd := exec.Command("pgrep", "-f", "xray.*confdir")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("未找到 Xray 进程，请检查服务状态")
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return fmt.Errorf("未找到运行中的 Xray 进程")
	}

	// 向所有 Xray 进程发送 USR1 信号
	for _, pid := range pids {
		killCmd := exec.Command("kill", "-USR1", pid)
		if err := killCmd.Run(); err != nil {
			utils.PrintWarning("向进程 %s 发送 USR1 信号失败: %v", pid, err)
		} else {
			utils.PrintSuccess("向进程 %s 发送热重载信号", pid)
		}
	}

	utils.PrintSuccess("配置热重载完成")
	return nil
}

// GenerateShareURL 生成协议分享链接
func (cm *ConfigManager) GenerateShareURL(tag string) (string, error) {
	info, err := cm.GetProtocolInfo(tag)
	if err != nil {
		return "", err
	}

	// 读取完整配置文件
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("protocol with tag '%s' not found", tag)
	}

	config, err := cm.readConfigFile(files[0].Path)
	if err != nil {
		return "", err
	}

	// 合并配置信息
	urlConfig := make(map[string]interface{})
	urlConfig["host"] = "localhost" // 默认主机，用户可以修改
	urlConfig["port"] = info.Port
	urlConfig["remark"] = tag

	// 从配置中提取所需信息
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok && len(inboundList) > 0 {
			if inbound, ok := inboundList[0].(map[string]interface{}); ok {
				// 提取端口
				if port, exists := inbound["port"]; exists {
					urlConfig["port"] = port
				}

				// 提取设置
				if settings, exists := inbound["settings"]; exists {
					if settingsMap, ok := settings.(map[string]interface{}); ok {
						// 提取客户端信息
						if clients, exists := settingsMap["clients"]; exists {
							if clientList, ok := clients.([]interface{}); ok && len(clientList) > 0 {
								if client, ok := clientList[0].(map[string]interface{}); ok {
									if uuid, exists := client["id"]; exists {
										urlConfig["uuid"] = uuid
									}
									if password, exists := client["password"]; exists {
										urlConfig["password"] = password
									}
								}
							}
						}

						// 提取加密方法（Shadowsocks）
						if method, exists := settingsMap["method"]; exists {
							urlConfig["method"] = method
						}
						if password, exists := settingsMap["password"]; exists {
							urlConfig["password"] = password
						}
					}
				}

				// 提取流设置
				if streamSettings, exists := inbound["streamSettings"]; exists {
					if streamMap, ok := streamSettings.(map[string]interface{}); ok {
						if network, exists := streamMap["network"]; exists {
							urlConfig["network"] = network
						}

						// WebSocket 设置
						if wsSettings, exists := streamMap["wsSettings"]; exists {
							if wsMap, ok := wsSettings.(map[string]interface{}); ok {
								if path, exists := wsMap["path"]; exists {
									urlConfig["path"] = path
								}
							}
						}

						// HTTPUpgrade 设置
						if huSettings, exists := streamMap["httpupgradeSettings"]; exists {
							if huMap, ok := huSettings.(map[string]interface{}); ok {
								if path, exists := huMap["path"]; exists {
									urlConfig["path"] = path
								}
							}
						}

						// REALITY 设置
						if realitySettings, exists := streamMap["realitySettings"]; exists {
							if realityMap, ok := realitySettings.(map[string]interface{}); ok {
								if dest, exists := realityMap["dest"]; exists {
									urlConfig["dest"] = dest
								}
								if publicKey, exists := realityMap["publicKey"]; exists {
									urlConfig["publicKey"] = publicKey
								}
								if serverNames, exists := realityMap["serverNames"]; exists {
									if serverNameList, ok := serverNames.([]interface{}); ok && len(serverNameList) > 0 {
										urlConfig["serverName"] = serverNameList[0]
									}
								}
								if shortIds, exists := realityMap["shortIds"]; exists {
									if shortIdList, ok := shortIds.([]interface{}); ok && len(shortIdList) > 0 {
										urlConfig["shortId"] = shortIdList[0]
									}
								}
								if fingerprint, exists := realityMap["fingerprint"]; exists {
									urlConfig["fingerprint"] = fingerprint
								}
							}
						}

						// TLS 设置
						if security, exists := streamMap["security"]; exists {
							urlConfig["security"] = security
						}
					}
				}
			}
		}
	}

	return utils.GenerateProtocolURL(info.Type, tag, urlConfig)
}

// BackupConfig 备份配置
func (cm *ConfigManager) BackupConfig(backupPath string) error {
	if backupPath == "" {
		// 生成默认备份路径
		timestamp := utils.GetCurrentTime()
		timestamp = strings.ReplaceAll(timestamp, " ", "_")
		timestamp = strings.ReplaceAll(timestamp, ":", "-")
		backupPath = fmt.Sprintf("xrf-backup-%s.tar.gz", timestamp)
	}

	// 检查配置目录是否存在
	if _, err := os.Stat(cm.confDir); os.IsNotExist(err) {
		return fmt.Errorf("configuration directory does not exist: %s", cm.confDir)
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "xrf-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建备份信息文件
	backupInfo := map[string]interface{}{
		"version":     "1.0.0",
		"backup_time": utils.GetCurrentTime(),
		"config_dir":  cm.confDir,
		"protocols":   []string{},
	}

	// 列出所有协议
	protocols, err := cm.ListProtocols()
	if err != nil {
		return fmt.Errorf("failed to list protocols: %w", err)
	}

	protocolTags := make([]string, len(protocols))
	for i, p := range protocols {
		protocolTags[i] = p.Tag
	}
	backupInfo["protocols"] = protocolTags

	// 写入备份信息
	infoPath := filepath.Join(tempDir, "backup-info.json")
	infoBytes, _ := json.MarshalIndent(backupInfo, "", "  ")
	if err := os.WriteFile(infoPath, infoBytes, 0644); err != nil {
		return fmt.Errorf("failed to write backup info: %w", err)
	}

	// 复制配置文件
	configBackupDir := filepath.Join(tempDir, "confs")
	if err := os.MkdirAll(configBackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create config backup dir: %w", err)
	}

	if err := copyDir(cm.confDir, configBackupDir); err != nil {
		return fmt.Errorf("failed to copy configuration files: %w", err)
	}

	// 创建 tar.gz 压缩包
	if err := createTarGz(tempDir, backupPath); err != nil {
		return fmt.Errorf("failed to create backup archive: %w", err)
	}

	utils.Success("Configuration backed up to: %s", backupPath)
	return nil
}

// RestoreConfig 恢复配置
func (cm *ConfigManager) RestoreConfig(backupPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "xrf-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压备份文件
	if err := extractTarGz(backupPath, tempDir); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	// 读取备份信息
	infoPath := filepath.Join(tempDir, "backup-info.json")
	infoBytes, err := os.ReadFile(infoPath)
	if err != nil {
		return fmt.Errorf("failed to read backup info: %w", err)
	}

	var backupInfo map[string]interface{}
	if err := json.Unmarshal(infoBytes, &backupInfo); err != nil {
		return fmt.Errorf("failed to parse backup info: %w", err)
	}

	// 备份当前配置（以防恢复失败）
	timestamp := utils.GetCurrentTime()
	timestamp = strings.ReplaceAll(timestamp, " ", "_")
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	currentBackupPath := fmt.Sprintf("xrf-current-backup-%s.tar.gz", timestamp)

	utils.Info("Creating backup of current configuration...")
	if err := cm.BackupConfig(currentBackupPath); err != nil {
		utils.Warning("Failed to backup current configuration: %v", err)
	}

	// 删除当前配置目录（如果存在）
	if _, err := os.Stat(cm.confDir); err == nil {
		if err := os.RemoveAll(cm.confDir); err != nil {
			return fmt.Errorf("failed to remove current configuration: %w", err)
		}
	}

	// 恢复配置文件
	configRestoreDir := filepath.Join(tempDir, "confs")
	if err := copyDir(configRestoreDir, cm.confDir); err != nil {
		return fmt.Errorf("failed to restore configuration files: %w", err)
	}

	// 显示恢复信息
	if backupTime, exists := backupInfo["backup_time"]; exists {
		utils.Success("Configuration restored from backup created at: %v", backupTime)
	}
	if protocols, exists := backupInfo["protocols"]; exists {
		if protocolList, ok := protocols.([]interface{}); ok {
			utils.Info("Restored %d protocol configurations", len(protocolList))
		}
	}

	// 清理临时备份文件（恢复成功后）
	if currentBackupPath != "" {
		if err := os.Remove(currentBackupPath); err != nil {
			utils.Warning("Failed to cleanup temporary backup file: %v", err)
		}
	}

	return nil
}

// 辅助函数

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if err := dstFile.Chmod(info.Mode()); err != nil {
			return err
		}

		_, err = srcFile.WriteTo(dstFile)
		return err
	})
}

func createTarGz(srcDir, dstPath string) error {
	// 使用系统 tar 命令创建压缩包
	cmd := exec.Command("tar", "-czf", dstPath, "-C", srcDir, ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar command failed: %w", err)
	}
	return nil
}

func extractTarGz(srcPath, dstDir string) error {
	// 使用系统 tar 命令解压
	cmd := exec.Command("tar", "-xzf", srcPath, "-C", dstDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extraction failed: %w", err)
	}
	return nil
}

// 生成模板数据
func (cm *ConfigManager) generateTemplateData(protocol Protocol, tag string, options map[string]interface{}) (TemplateData, error) {
	data := TemplateData{
		Tag: tag,
	}

	// 设置端口（包含端口冲突检查）
	if port, exists := options["port"]; exists {
		if portInt, ok := port.(int); ok {
			data.Port = portInt
		} else if portStr, ok := port.(string); ok {
			if portInt, err := strconv.Atoi(portStr); err == nil {
				data.Port = portInt
			}
		}
	}

	// 智能端口分配
	if data.Port == 0 {
		// 检查默认端口是否可用
		if utils.IsPortAvailable(protocol.DefaultPort) {
			data.Port = protocol.DefaultPort
		} else {
			// 建议替代端口
			suggestedPort, err := utils.SuggestPort(protocol.Name, 0)
			if err != nil {
				utils.PrintWarning("无法找到可用端口: %v", err)
				data.Port = protocol.DefaultPort // 回退到默认端口
			} else {
				data.Port = suggestedPort
				utils.PrintInfo("端口 %d 已被占用，自动分配端口 %d", protocol.DefaultPort, suggestedPort)
			}
		}
	} else {
		// 验证指定端口是否可用
		if !utils.IsPortAvailable(data.Port) {
			return data, fmt.Errorf("端口 %d 已被占用，请选择其他端口", data.Port)
		}
	}

	// 生成 UUID
	if uuid, exists := options["uuid"]; exists {
		if uuidStr, ok := uuid.(string); ok {
			data.UUID = uuidStr
		}
	}
	if data.UUID == "" && (strings.Contains(protocol.Name, "VLESS") || strings.Contains(protocol.Name, "VMess")) {
		data.UUID = utils.GenerateUUID()
	}

	// 设置加密方法（Shadowsocks）
	if method, exists := options["method"]; exists {
		if methodStr, ok := method.(string); ok {
			data.Method = methodStr
		}
	}
	if data.Method == "" && strings.Contains(protocol.Name, "Shadowsocks") {
		if strings.Contains(protocol.Name, "Shadowsocks-2022") {
			data.Method = "2022-blake3-aes-256-gcm"
		} else {
			data.Method = "chacha20-poly1305"
		}
	}

	// 生成密码
	if password, exists := options["password"]; exists {
		if passwordStr, ok := password.(string); ok {
			data.Password = passwordStr
		}
	}
	if data.Password == "" && (strings.Contains(protocol.Name, "Trojan") || strings.Contains(protocol.Name, "Shadowsocks")) {
		if strings.Contains(protocol.Name, "Shadowsocks-2022") && data.Method != "" {
			if key, err := utils.GenerateShadowsocks2022Key(data.Method); err == nil {
				data.Password = key
			}
		}
		if data.Password == "" {
			data.Password = utils.GeneratePassword(16)
		}
	}

	// 设置路径
	if path, exists := options["path"]; exists {
		if pathStr, ok := path.(string); ok {
			data.Path = pathStr
		}
	}
	if data.Path == "" {
		data.Path = "/ws"
	}

	// 设置域名
	if host, exists := options["host"]; exists {
		if hostStr, ok := host.(string); ok {
			data.Host = hostStr
		}
	}

	// REALITY 特定设置
	if strings.Contains(protocol.Name, "REALITY") {
		if dest, exists := options["dest"]; exists {
			if destStr, ok := dest.(string); ok {
				data.Dest = destStr
			}
		} else if domain, exists := options["domain"]; exists {
			// 兼容处理误用的domain参数
			if domainStr, ok := domain.(string); ok {
				data.Dest = domainStr
				utils.PrintWarning("VLESS-REALITY不需要域名证书，已将 '%s' 作为伪装目标使用", domainStr)
				utils.PrintInfo("建议使用: --dest %s", domainStr)
			}
		}
		if data.Dest == "" {
			data.Dest = "www.microsoft.com"
		}

		if serverName, exists := options["serverName"]; exists {
			if serverNameStr, ok := serverName.(string); ok {
				data.ServerName = serverNameStr
			}
		}
		if data.ServerName == "" {
			data.ServerName = data.Dest
		}

		// 生成密钥对
		if privateKey, exists := options["privateKey"]; exists {
			if privateKeyStr, ok := privateKey.(string); ok {
				data.PrivateKey = privateKeyStr
			}
		}
		if data.PrivateKey == "" {
			if priv, _, err := utils.GenerateX25519KeyPair(); err == nil {
				data.PrivateKey = priv
			}
		}

		if shortId, exists := options["shortId"]; exists {
			if shortIdStr, ok := shortId.(string); ok {
				data.ShortId = shortIdStr
			}
		}
		if data.ShortId == "" {
			data.ShortId = utils.GenerateShortID(8)
		}
	}

	// TLS 设置
	if protocol.RequiresTLS {
		data.Security = "tls"

		// 测试环境证书处理
		if IsTestEnvironment() {
			// 检查是否手动提供了证书
			_, hasCert := options["certFile"]
			_, hasKey := options["keyFile"]

			if !hasCert && !hasKey {
				// 使用测试证书
				testCert, err := GetTestCertificate()
				if err != nil {
					return data, fmt.Errorf("failed to get test certificate: %v", err)
				}
				data.CertFile = testCert.CertFile
				data.KeyFile = testCert.KeyFile
			} else {
				// 使用手动提供的证书
				if hasCert {
					if certFileStr, ok := options["certFile"].(string); ok {
						data.CertFile = certFileStr
					}
				}
				if hasKey {
					if keyFileStr, ok := options["keyFile"].(string); ok {
						data.KeyFile = keyFileStr
					}
				}
			}
		} else {
			// 生产环境证书处理（现有逻辑）
			if certFile, exists := options["certFile"]; exists {
				if certFileStr, ok := certFile.(string); ok {
					data.CertFile = certFileStr
				}
			}
			if keyFile, exists := options["keyFile"]; exists {
				if keyFileStr, ok := keyFile.(string); ok {
					data.KeyFile = keyFileStr
				}
			}
		}
	}

	return data, nil
}

// 生成配置文件名
func (cm *ConfigManager) generateConfigFilename(configType, name string, isTail bool) string {
	priority := cm.getConfigPriority(configType, name)

	if isTail {
		return fmt.Sprintf("%02d-%s-%s-tail.json", priority, configType, name)
	}
	return fmt.Sprintf("%02d-%s-%s.json", priority, configType, name)
}

// 获取配置优先级
func (cm *ConfigManager) getConfigPriority(configType, name string) int {
	priorityMap := map[string]map[string]int{
		"base":     {"system": 0},
		"dns":      {"default": 1},
		"inbound":  {"default": 10},
		"outbound": {"direct": 20, "block": 21, "default": 25},
		"routing":  {"basic": 90, "default": 91, "tail": 99},
	}

	if typeMap, exists := priorityMap[configType]; exists {
		if priority, exists := typeMap[name]; exists {
			return priority
		}
		if priority, exists := typeMap["default"]; exists {
			return priority
		}
	}
	return 50
}

// 列出配置文件
func (cm *ConfigManager) listConfigFiles() ([]ConfigFile, error) {
	files, err := os.ReadDir(cm.confDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configFiles []ConfigFile
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		configFile := cm.parseConfigFileName(file.Name())
		configFile.Path = filepath.Join(cm.confDir, file.Name())
		configFiles = append(configFiles, configFile)
	}

	// 按优先级排序
	sort.Slice(configFiles, func(i, j int) bool {
		return configFiles[i].Priority < configFiles[j].Priority
	})

	return configFiles, nil
}

// 解析配置文件名
func (cm *ConfigManager) parseConfigFileName(filename string) ConfigFile {
	// 格式: 00-type-name.json 或 99-type-name-tail.json
	name := strings.TrimSuffix(filename, ".json")
	parts := strings.Split(name, "-")

	if len(parts) < 3 {
		return ConfigFile{Filename: filename}
	}

	priority, _ := strconv.Atoi(parts[0])
	configType := parts[1]
	configName := strings.Join(parts[2:], "-")

	isTail := strings.HasSuffix(configName, "-tail")
	if isTail {
		configName = strings.TrimSuffix(configName, "-tail")
	}

	return ConfigFile{
		Priority: priority,
		Type:     configType,
		Name:     configName,
		Filename: filename,
		IsTail:   isTail,
	}
}

// 根据 tag 查找配置文件
func (cm *ConfigManager) findConfigFilesByTag(tag string) ([]ConfigFile, error) {
	files, err := cm.listConfigFiles()
	if err != nil {
		return nil, err
	}

	var matchingFiles []ConfigFile
	for _, file := range files {
		config, err := cm.readConfigFile(file.Path)
		if err != nil {
			continue
		}

		if cm.configContainsTag(config, tag) {
			matchingFiles = append(matchingFiles, file)
		}
	}

	return matchingFiles, nil
}

// 检查配置是否包含指定 tag
func (cm *ConfigManager) configContainsTag(config map[string]interface{}, tag string) bool {
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok {
			for _, inbound := range inboundList {
				if inboundMap, ok := inbound.(map[string]interface{}); ok {
					if inboundTag, exists := inboundMap["tag"]; exists {
						if inboundTag == tag {
							return true
						}
					}
				}
			}
		}
	}

	if outbounds, exists := config["outbounds"]; exists {
		if outboundList, ok := outbounds.([]interface{}); ok {
			for _, outbound := range outboundList {
				if outboundMap, ok := outbound.(map[string]interface{}); ok {
					if outboundTag, exists := outboundMap["tag"]; exists {
						if outboundTag == tag {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// 读取配置文件
func (cm *ConfigManager) readConfigFile(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// 解析协议信息
func (cm *ConfigManager) parseProtocolInfo(file ConfigFile) (ProtocolInfo, error) {
	config, err := cm.readConfigFile(file.Path)
	if err != nil {
		return ProtocolInfo{}, err
	}

	info := ProtocolInfo{
		Tag:        file.Name,
		ConfigFile: file.Filename,
		Settings:   make(map[string]interface{}),
		Status:     "unknown",
	}

	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok && len(inboundList) > 0 {
			if inbound, ok := inboundList[0].(map[string]interface{}); ok {
				if protocol, exists := inbound["protocol"]; exists {
					info.Type = fmt.Sprintf("%v", protocol)
				}
				if port, exists := inbound["port"]; exists {
					if portInt, ok := port.(float64); ok {
						info.Port = int(portInt)
					}
				}
				if tag, exists := inbound["tag"]; exists {
					info.Tag = fmt.Sprintf("%v", tag)
				}
				info.Settings = inbound
			}
		}
	}

	return info, nil
}

// GetProtocolInfo 获取协议详细信息
func (cm *ConfigManager) GetProtocolInfo(tag string) (ProtocolInfo, error) {
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		return ProtocolInfo{}, err
	}

	if len(files) == 0 {
		return ProtocolInfo{}, fmt.Errorf("protocol with tag '%s' not found", tag)
	}

	// 使用第一个匹配的文件
	file := files[0]
	info, err := cm.parseProtocolInfo(file)
	if err != nil {
		return ProtocolInfo{}, err
	}

	// 获取协议定义以补充更多信息
	if protocol, err := cm.protocols.GetProtocol(info.Type); err == nil {
		info.Settings["description"] = protocol.Description
		info.Settings["aliases"] = protocol.Aliases
		info.Settings["requiresTLS"] = protocol.RequiresTLS
		info.Settings["requiresDomain"] = protocol.RequiresDomain
		info.Settings["supportedTransports"] = protocol.SupportedTransports
	}

	return info, nil
}

// 合并配置选项
func (cm *ConfigManager) mergeConfigOptions(config map[string]interface{}, options map[string]interface{}) error {
	// 递归合并配置
	for key, newValue := range options {
		switch key {
		case "port":
			// 更新入站端口
			if err := cm.updateInboundPort(config, newValue); err != nil {
				return err
			}
		case "password":
			// 更新密码
			if err := cm.updatePassword(config, newValue); err != nil {
				return err
			}
		case "uuid":
			// 更新 UUID
			if err := cm.updateUUID(config, newValue); err != nil {
				return err
			}
		case "path":
			// 更新路径
			if err := cm.updateTransportPath(config, newValue); err != nil {
				return err
			}
		default:
			// 其他简单键值对直接更新
			config[key] = newValue
		}
	}

	return nil
}

// 更新入站端口
func (cm *ConfigManager) updateInboundPort(config map[string]interface{}, portValue interface{}) error {
	var port int

	switch v := portValue.(type) {
	case int:
		port = v
	case string:
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		} else {
			return fmt.Errorf("invalid port format: %s", v)
		}
	case float64:
		port = int(v)
	default:
		return fmt.Errorf("unsupported port type: %T", portValue)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	// 更新 inbounds 中的端口
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok {
			for _, inbound := range inboundList {
				if inboundMap, ok := inbound.(map[string]interface{}); ok {
					inboundMap["port"] = port
				}
			}
		}
	}

	return nil
}

// 更新密码
func (cm *ConfigManager) updatePassword(config map[string]interface{}, passwordValue interface{}) error {
	password, ok := passwordValue.(string)
	if !ok {
		return fmt.Errorf("password must be a string")
	}

	// 更新 inbounds 中的密码
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok {
			for _, inbound := range inboundList {
				if inboundMap, ok := inbound.(map[string]interface{}); ok {
					if settings, exists := inboundMap["settings"]; exists {
						if settingsMap, ok := settings.(map[string]interface{}); ok {
							// Shadowsocks
							if _, exists := settingsMap["method"]; exists {
								settingsMap["password"] = password
							}
							// Trojan
							if clients, exists := settingsMap["clients"]; exists {
								if clientList, ok := clients.([]interface{}); ok {
									for _, client := range clientList {
										if clientMap, ok := client.(map[string]interface{}); ok {
											clientMap["password"] = password
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// 更新 UUID
func (cm *ConfigManager) updateUUID(config map[string]interface{}, uuidValue interface{}) error {
	uuid, ok := uuidValue.(string)
	if !ok {
		return fmt.Errorf("uuid must be a string")
	}

	// 验证 UUID 格式
	if !utils.IsValidUUID(uuid) {
		return fmt.Errorf("invalid UUID format")
	}

	// 更新 inbounds 中的 UUID
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok {
			for _, inbound := range inboundList {
				if inboundMap, ok := inbound.(map[string]interface{}); ok {
					if settings, exists := inboundMap["settings"]; exists {
						if settingsMap, ok := settings.(map[string]interface{}); ok {
							if clients, exists := settingsMap["clients"]; exists {
								if clientList, ok := clients.([]interface{}); ok {
									for _, client := range clientList {
										if clientMap, ok := client.(map[string]interface{}); ok {
											clientMap["id"] = uuid
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// 更新传输路径
func (cm *ConfigManager) updateTransportPath(config map[string]interface{}, pathValue interface{}) error {
	path, ok := pathValue.(string)
	if !ok {
		return fmt.Errorf("path must be a string")
	}

	if path == "" || !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}

	// 更新 inbounds 中的路径
	if inbounds, exists := config["inbounds"]; exists {
		if inboundList, ok := inbounds.([]interface{}); ok {
			for _, inbound := range inboundList {
				if inboundMap, ok := inbound.(map[string]interface{}); ok {
					if streamSettings, exists := inboundMap["streamSettings"]; exists {
						if streamMap, ok := streamSettings.(map[string]interface{}); ok {
							// WebSocket
							if wsSettings, exists := streamMap["wsSettings"]; exists {
								if wsMap, ok := wsSettings.(map[string]interface{}); ok {
									wsMap["path"] = path
								}
							}
							// HTTPUpgrade
							if httpupgradeSettings, exists := streamMap["httpupgradeSettings"]; exists {
								if httpupgradeMap, ok := httpupgradeSettings.(map[string]interface{}); ok {
									httpupgradeMap["path"] = path
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Enhanced rollback mechanism helper methods (DESIGN.md requirement line 575)

// createAutoBackup creates an automatic backup before configuration changes
func (cm *ConfigManager) createAutoBackup(operation string) (string, error) {
	timestamp := utils.GetCurrentTime()
	timestamp = strings.ReplaceAll(timestamp, " ", "_")
	timestamp = strings.ReplaceAll(timestamp, ":", "-")
	backupPath := fmt.Sprintf("/tmp/xrf-auto-backup-%s-%s.tar.gz", operation, timestamp)

	if err := cm.BackupConfig(backupPath); err != nil {
		return "", fmt.Errorf("failed to create auto backup: %w", err)
	}

	return backupPath, nil
}

// restoreFromBackup restores configuration from backup file
func (cm *ConfigManager) restoreFromBackup(backupPath string) error {
	return cm.RestoreConfig(backupPath)
}

// validateConfigAfterChange validates configuration after making changes
func (cm *ConfigManager) validateConfigAfterChange() error {
	// Use xray run -test command to validate configuration
	cmd := exec.Command("xray", "run", "-test", "-confdir", cm.confDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("configuration validation failed: %s", string(output))
	}
	return nil
}
