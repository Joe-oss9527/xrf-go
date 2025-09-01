package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/yourusername/xrf-go/pkg/utils"
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
		
		if err := ioutil.WriteFile(configPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", config.filename, err)
		}
		
		utils.Debug("Created base config: %s", config.filename)
	}
	
	return nil
}

// AddProtocol 添加协议配置
func (cm *ConfigManager) AddProtocol(protocolType, tag string, options map[string]interface{}) error {
	protocol, err := cm.protocols.GetProtocol(protocolType)
	if err != nil {
		return err
	}
	
	// 生成模板数据
	data, err := cm.generateTemplateData(protocol, tag, options)
	if err != nil {
		return fmt.Errorf("failed to generate template data: %w", err)
	}
	
	// 获取协议模板
	templateStr, exists := GetProtocolTemplate(protocol.Template)
	if !exists {
		return fmt.Errorf("template not found for protocol: %s", protocol.Template)
	}
	
	// 渲染配置模板
	content, err := cm.renderer.Render(templateStr, data)
	if err != nil {
		return fmt.Errorf("failed to render protocol template: %w", err)
	}
	
	// 生成配置文件名
	filename := cm.generateConfigFilename("inbound", tag, false)
	configPath := filepath.Join(cm.confDir, filename)
	
	// 写入配置文件
	if err := ioutil.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write protocol config: %w", err)
	}
	
	utils.Success("Added protocol %s with tag %s", protocol.Name, tag)
	return nil
}

// RemoveProtocol 删除协议配置
func (cm *ConfigManager) RemoveProtocol(tag string) error {
	// 查找包含该 tag 的配置文件
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		return err
	}
	
	if len(files) == 0 {
		return fmt.Errorf("protocol with tag '%s' not found", tag)
	}
	
	// 删除配置文件
	for _, file := range files {
		if err := os.Remove(file.Path); err != nil {
			utils.Warning("Failed to remove config file %s: %v", file.Filename, err)
		} else {
			utils.Debug("Removed config file: %s", file.Filename)
		}
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

// UpdateProtocol 更新协议配置
func (cm *ConfigManager) UpdateProtocol(tag string, options map[string]interface{}) error {
	// 查找现有配置
	files, err := cm.findConfigFilesByTag(tag)
	if err != nil {
		return err
	}
	
	if len(files) == 0 {
		return fmt.Errorf("protocol with tag '%s' not found", tag)
	}
	
	// 读取现有配置
	configFile := files[0]
	existingConfig, err := cm.readConfigFile(configFile.Path)
	if err != nil {
		return err
	}
	
	// 更新配置
	if err := cm.mergeConfigOptions(existingConfig, options); err != nil {
		return err
	}
	
	// 写回配置文件
	updatedContent, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}
	
	if err := ioutil.WriteFile(configFile.Path, updatedContent, 0644); err != nil {
		return fmt.Errorf("failed to write updated config: %w", err)
	}
	
	utils.Success("Updated protocol %s", tag)
	return nil
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig() error {
	return utils.ValidateXrayConfDir(cm.confDir)
}

// BackupConfig 备份配置
func (cm *ConfigManager) BackupConfig(backupPath string) error {
	// TODO: 实现配置备份
	return fmt.Errorf("backup functionality not implemented yet")
}

// RestoreConfig 恢复配置
func (cm *ConfigManager) RestoreConfig(backupPath string) error {
	// TODO: 实现配置恢复
	return fmt.Errorf("restore functionality not implemented yet")
}

// 生成模板数据
func (cm *ConfigManager) generateTemplateData(protocol Protocol, tag string, options map[string]interface{}) (TemplateData, error) {
	data := TemplateData{
		Tag: tag,
	}
	
	// 设置端口
	if port, exists := options["port"]; exists {
		if portInt, ok := port.(int); ok {
			data.Port = portInt
		} else if portStr, ok := port.(string); ok {
			if portInt, err := strconv.Atoi(portStr); err == nil {
				data.Port = portInt
			}
		}
	}
	if data.Port == 0 {
		data.Port = protocol.DefaultPort
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
	files, err := ioutil.ReadDir(cm.confDir)
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
	content, err := ioutil.ReadFile(path)
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

// 合并配置选项
func (cm *ConfigManager) mergeConfigOptions(config map[string]interface{}, options map[string]interface{}) error {
	// TODO: 实现配置选项合并逻辑
	return fmt.Errorf("config merging not implemented yet")
}