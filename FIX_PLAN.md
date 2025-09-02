# XRF-Go Systemd GROUP错误修复计划

## 问题分析

### 1. 核心问题
- **硬编码的 `nobody/nogroup` 在不同Linux发行版上不兼容**
  - Rocky/RHEL/CentOS: 只有 `nobody` 组，没有 `nogroup`
  - Ubuntu/Debian: 有 `nobody` 用户和 `nogroup` 组
  - 其他发行版: 可能都没有

### 2. 代码位置
- `pkg/system/service.go:17-18`: 硬编码的用户/组常量
- `pkg/system/service.go:496-505`: 只创建用户，未处理组
- `pkg/system/service.go:81-114`: systemd服务文件生成

### 3. 影响范围
- 所有服务安装、启动、重启操作
- 测试文件也使用了相同的常量

## 修复方案

### 阶段1: 创建专用服务用户（推荐）

#### 1.1 修改常量定义 (`pkg/system/service.go:14-19`)
```go
const (
    ServiceName        = "xray"
    SystemdServicePath = "/etc/systemd/system/xray.service"
    ServiceUser        = "xray"      // 改为专用用户
    ServiceGroup       = "xray"      // 改为专用组
    ServiceHome        = "/var/lib/xray"
    ServiceShell       = "/usr/sbin/nologin"
)
```

#### 1.2 实现完整的用户/组管理 (`pkg/system/service.go:489-522`)
```go
// ensureServiceUserAndGroup 确保服务用户和组存在
func (sm *ServiceManager) ensureServiceUserAndGroup() error {
    // 1. 检查并创建组
    if err := exec.Command("getent", "group", ServiceGroup).Run(); err != nil {
        // 创建系统组
        if err := exec.Command("groupadd", "--system", ServiceGroup).Run(); err != nil {
            // 尝试不同的命令格式（兼容不同发行版）
            if err := exec.Command("groupadd", "-r", ServiceGroup).Run(); err != nil {
                return fmt.Errorf("failed to create group %s: %w", ServiceGroup, err)
            }
        }
        utils.PrintInfo("Created system group: %s", ServiceGroup)
    }
    
    // 2. 检查并创建用户
    if err := exec.Command("getent", "passwd", ServiceUser).Run(); err != nil {
        // 检测 shell 路径
        shell := ServiceShell
        if _, err := os.Stat(shell); err != nil {
            // 尝试其他常见路径
            for _, altShell := range []string{"/sbin/nologin", "/bin/false"} {
                if _, err := os.Stat(altShell); err == nil {
                    shell = altShell
                    break
                }
            }
        }
        
        // 创建系统用户
        cmd := exec.Command("useradd",
            "--system",
            "--gid", ServiceGroup,
            "--home-dir", ServiceHome,
            "--no-create-home",
            "--shell", shell,
            "--comment", "Xray proxy service user",
            ServiceUser)
        
        if err := cmd.Run(); err != nil {
            // 尝试简化版本
            cmd = exec.Command("useradd", "-r", "-g", ServiceGroup, "-s", shell, ServiceUser)
            if err := cmd.Run(); err != nil {
                return fmt.Errorf("failed to create user %s: %w", ServiceUser, err)
            }
        }
        utils.PrintInfo("Created system user: %s", ServiceUser)
    }
    
    return nil
}
```

#### 1.3 增强服务安装流程 (`pkg/system/service.go:45-78`)
```go
func (sm *ServiceManager) InstallService() error {
    if !sm.detector.IsRoot() {
        return fmt.Errorf("需要 root 权限才能安装服务")
    }

    sysInfo := sm.detector.GetSystemInfo()
    if sysInfo != nil && !sysInfo.HasSystemd {
        return fmt.Errorf("系统不支持 systemd")
    }

    utils.PrintInfo("安装 Xray 服务...")

    // 确保服务用户和组存在
    if err := sm.ensureServiceUserAndGroup(); err != nil {
        return fmt.Errorf("创建服务用户失败: %w", err)
    }

    // 生成服务文件内容
    serviceContent := sm.generateServiceFile()

    // 写入服务文件
    if err := os.WriteFile(SystemdServicePath, []byte(serviceContent), 0644); err != nil {
        return fmt.Errorf("写入服务文件失败: %w", err)
    }

    // 设置配置目录权限
    if err := sm.setDirectoryPermissions(); err != nil {
        utils.PrintWarning("设置目录权限失败: %v", err)
    }

    // 重新加载 systemd
    if err := sm.reloadSystemd(); err != nil {
        return fmt.Errorf("重新加载 systemd 失败: %w", err)
    }

    // 启用服务
    if err := sm.EnableService(); err != nil {
        return fmt.Errorf("启用服务失败: %w", err)
    }

    utils.PrintSuccess("Xray 服务安装完成")
    return nil
}
```

### 阶段2: 智能降级策略

#### 2.1 实现OS感知的用户选择
```go
// getSystemUserGroup 根据系统类型获取合适的用户和组
func (sm *ServiceManager) getSystemUserGroup() (user, group string) {
    // 优先使用专用用户
    if exec.Command("getent", "passwd", "xray").Run() == nil {
        return "xray", "xray"
    }
    
    // 基于发行版选择合适的降级方案
    sysInfo := sm.detector.GetSystemInfo()
    if sysInfo != nil {
        switch sysInfo.Distribution {
        case "rocky", "rhel", "centos", "fedora", "almalinux":
            // RHEL系列使用 nobody/nobody
            if exec.Command("getent", "group", "nobody").Run() == nil {
                return "nobody", "nobody"
            }
        case "ubuntu", "debian", "raspbian":
            // Debian系列使用 nobody/nogroup
            if exec.Command("getent", "group", "nogroup").Run() == nil {
                return "nobody", "nogroup"
            }
        case "alpine":
            // Alpine Linux
            return "nobody", "nobody"
        }
    }
    
    // 默认创建专用用户
    if err := sm.ensureServiceUserAndGroup(); err == nil {
        return ServiceUser, ServiceGroup
    }
    
    // 最后的降级选项
    return "nobody", "nobody"
}
```

#### 2.2 动态生成服务文件
```go
func (sm *ServiceManager) generateServiceFile() string {
    user, group := sm.getSystemUserGroup()
    
    return fmt.Sprintf(`[Unit]
Description=Xray Service
Documentation=https://github.com/XTLS/Xray-core
After=network.target nss-lookup.target

[Service]
Type=simple
User=%s
Group=%s
ExecStart=%s run -confdir %s
Restart=on-failure
RestartPreventExitStatus=1
RestartSec=5
LimitNOFILE=1000000

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=%s /var/lib/xray
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

# Network settings
PrivateDevices=true
RestrictSUIDSGID=true
RestrictRealtime=true
RestrictNamespaces=true

[Install]
WantedBy=multi-user.target
`, user, group, XrayBinaryPath, XrayConfsDir, filepath.Dir(XrayConfsDir))
}
```

### 阶段3: 增强测试覆盖

#### 3.1 添加用户管理单元测试
```go
func TestServiceUserManagement(t *testing.T) {
    t.Parallel()
    
    detector := NewDetector()
    sm := NewServiceManager(detector)
    
    // 测试获取系统用户组
    user, group := sm.getSystemUserGroup()
    if user == "" || group == "" {
        t.Error("getSystemUserGroup() returned empty user or group")
    }
    
    // 验证用户组合理性
    validUsers := []string{"xray", "nobody"}
    validGroups := []string{"xray", "nobody", "nogroup"}
    
    userValid := false
    for _, v := range validUsers {
        if user == v {
            userValid = true
            break
        }
    }
    
    groupValid := false
    for _, v := range validGroups {
        if group == v {
            groupValid = true
            break
        }
    }
    
    if !userValid {
        t.Errorf("Invalid user: %s", user)
    }
    if !groupValid {
        t.Errorf("Invalid group: %s", group)
    }
}
```

### 阶段4: 添加卸载清理功能

#### 4.1 增强卸载命令
```go
func (sm *ServiceManager) UninstallService(purgeUser bool) error {
    if !sm.detector.IsRoot() {
        return fmt.Errorf("需要 root 权限才能卸载服务")
    }

    utils.PrintInfo("卸载 Xray 服务...")

    // 停止服务
    sm.StopService() // 忽略错误

    // 禁用服务
    sm.DisableService() // 忽略错误

    // 删除服务文件
    if err := os.Remove(SystemdServicePath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("删除服务文件失败: %w", err)
    }

    // 重新加载 systemd
    if err := sm.reloadSystemd(); err != nil {
        utils.PrintWarning("重新加载 systemd 失败: %v", err)
    }

    // 可选：删除服务用户和组
    if purgeUser && ServiceUser == "xray" {
        sm.removeServiceUser()
    }

    utils.PrintSuccess("Xray 服务已卸载")
    return nil
}

func (sm *ServiceManager) removeServiceUser() {
    // 删除用户
    if exec.Command("getent", "passwd", ServiceUser).Run() == nil {
        if err := exec.Command("userdel", ServiceUser).Run(); err != nil {
            utils.PrintWarning("删除用户 %s 失败: %v", ServiceUser, err)
        } else {
            utils.PrintInfo("已删除用户: %s", ServiceUser)
        }
    }
    
    // 删除组
    if exec.Command("getent", "group", ServiceGroup).Run() == nil {
        if err := exec.Command("groupdel", ServiceGroup).Run(); err != nil {
            utils.PrintWarning("删除组 %s 失败: %v", ServiceGroup, err)
        } else {
            utils.PrintInfo("已删除组: %s", ServiceGroup)
        }
    }
}
```

## 实施步骤

1. **第一步**: 修改服务常量和添加用户管理函数
   - 文件: `pkg/system/service.go`
   - 添加 `ensureServiceUserAndGroup()` 方法
   - 添加 `getSystemUserGroup()` 方法
   - 修改 `InstallService()` 调用新方法

2. **第二步**: 更新测试文件
   - 文件: `pkg/system/service_test.go`
   - 文件: `pkg/system/service_unit_test.go`
   - 更新测试以适应新的用户/组

3. **第三步**: 增强卸载功能
   - 修改 `UninstallService()` 添加 purgeUser 参数
   - 在命令行添加 `--purge-user` 选项

4. **第四步**: 更新文档
   - 更新 README.md 说明新的用户管理机制
   - 添加升级指南

## 测试计划

### 环境准备
```bash
# Rocky Linux 9
docker run -it --rm rockylinux:9 bash

# Ubuntu 22.04
docker run -it --rm ubuntu:22.04 bash

# Debian 12
docker run -it --rm debian:12 bash
```

### 测试用例
1. **全新安装测试**
   - 运行 `xrf install`
   - 验证服务启动成功
   - 检查用户/组创建: `id xray`

2. **服务操作测试**
   - `xrf start`
   - `xrf stop`
   - `xrf restart`
   - `xrf status`

3. **权限测试**
   - 检查配置目录权限: `ls -la /etc/xray/`
   - 验证服务以正确用户运行: `ps aux | grep xray`

4. **卸载测试**
   - `xrf uninstall --purge-user`
   - 验证用户被删除: `id xray` (应该失败)

## 风险评估

### 风险点
1. **升级兼容性**: 现有安装使用 nobody 用户
2. **权限迁移**: 配置文件权限需要调整
3. **不同发行版差异**: shell路径、命令参数

### 缓解措施
1. **平滑升级**: 检测现有安装，保持兼容
2. **智能降级**: 创建失败时降级到 nobody
3. **详细日志**: 记录用户创建过程
4. **回滚机制**: 安装失败时清理已创建的用户

## 预期效果

1. **解决GROUP错误**: 所有主流Linux发行版正常运行
2. **提升安全性**: 专用服务账户，最小权限原则
3. **改善用户体验**: 清晰的错误提示和恢复建议
4. **保持兼容性**: 支持升级和降级场景

## 参考资料

- xray-fusion 实现: 专用 xray 用户/组
- systemd 最佳实践: [systemd.exec(5)](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- Linux 用户管理: useradd(8), groupadd(8)