package utils

import (
	"fmt"
	"strings"
)

// ErrorType 错误类型
type ErrorType int

const (
	// 配置相关错误
	ErrConfigNotFound ErrorType = iota
	ErrConfigInvalid
	ErrConfigConflict
	ErrProtocolNotSupported
	ErrPortInUse
	ErrDomainRequired
	ErrCertificateInvalid

	// 系统相关错误
	ErrSystemNotSupported
	ErrPermissionDenied
	ErrServiceNotRunning
	ErrInstallationFailed
	ErrNetworkUnavailable

	// 文件操作错误
	ErrFileNotFound
	ErrFilePermission
	ErrFileCorrupted
	ErrDiskSpaceInsufficient
)

// XRFError 自定义错误类型
type XRFError struct {
	Type        ErrorType
	Message     string
	Cause       error
	Suggestions []string
	Context     map[string]interface{}
}

func (e *XRFError) Error() string {
	return e.Message
}

func (e *XRFError) Unwrap() error {
	return e.Cause
}

// GetSuggestions 获取错误修复建议
func (e *XRFError) GetSuggestions() []string {
	return e.suggestions()
}

// GetFormattedError 获取格式化的错误信息
func (e *XRFError) GetFormattedError() string {
	var sb strings.Builder

	// 错误信息
	sb.WriteString(fmt.Sprintf("❌ %s\n", e.Message))

	// 原始错误
	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf("原因: %s\n", e.Cause.Error()))
	}

	// 修复建议
	suggestions := e.suggestions()
	if len(suggestions) > 0 {
		sb.WriteString("\n💡 修复建议:\n")
		for i, suggestion := range suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
	}

	// 上下文信息
	if len(e.Context) > 0 {
		sb.WriteString("\n📋 详细信息:\n")
		for key, value := range e.Context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	return sb.String()
}

// suggestions 根据错误类型生成修复建议
func (e *XRFError) suggestions() []string {
	switch e.Type {
	case ErrConfigNotFound:
		return []string{
			"运行 'xrf init' 初始化配置",
			"检查配置目录权限是否正确",
			"确认 XRF 已正确安装",
		}

	case ErrProtocolNotSupported:
		return []string{
			"运行 'xrf list --protocols' 查看支持的协议",
			"检查协议名拼写是否正确",
			"尝试使用协议别名，如 'vr' 代替 'vless-reality'",
		}

	case ErrPortInUse:
		return []string{
			"使用 'xrf check-port <端口>' 检查端口可用性",
			"尝试使用其他端口号",
			"停止占用该端口的服务",
			"运行 'netstat -tlnp | grep <端口>' 查找占用进程",
		}

	case ErrDomainRequired:
		return []string{
			"添加 '--domain' 参数指定域名",
			"确保域名已正确解析到服务器",
			"检查域名格式是否正确",
		}

	case ErrCertificateInvalid:
		return []string{
			"检查证书文件路径是否正确",
			"确认证书文件格式为 PEM",
			"验证证书是否已过期",
			"重新生成或更新证书文件",
		}

	case ErrSystemNotSupported:
		return []string{
			"检查操作系统是否为支持的 Linux 发行版",
			"确认系统架构为 amd64 或 arm64",
			"联系开发团队申请支持新系统",
		}

	case ErrPermissionDenied:
		return []string{
			"使用 'sudo' 运行命令",
			"检查当前用户权限",
			"确认对配置目录有读写权限",
		}

	case ErrServiceNotRunning:
		return []string{
			"运行 'xrf start' 启动服务",
			"检查服务状态: 'xrf status'",
			"查看服务日志: 'xrf logs'",
		}

	case ErrInstallationFailed:
		return []string{
			"检查网络连接是否正常",
			"确认有足够的磁盘空间",
			"尝试手动下载安装包",
			"检查防火墙设置",
		}

	case ErrFileNotFound:
		return []string{
			"检查文件路径是否正确",
			"确认文件是否存在",
			"检查文件权限设置",
		}

	case ErrConfigInvalid:
		return []string{
			"运行 'xrf test' 验证配置",
			"检查配置文件语法",
			"恢复备份配置: 'xrf restore'",
		}

	case ErrConfigConflict:
		return []string{
			"使用不同的标签名称",
			"运行 'xrf list' 查看现有协议",
			"删除冲突的协议: 'xrf remove <标签>'",
			"使用 'xrf change' 修改现有协议",
		}

	case ErrNetworkUnavailable:
		return []string{
			"检查网络连接",
			"验证 DNS 设置",
			"检查防火墙规则",
			"尝试使用代理访问",
		}

	default:
		return []string{
			"查看详细日志: 'xrf logs --error'",
			"运行 'xrf test' 进行诊断",
			"访问项目文档获取更多帮助",
		}
	}
}

// 便捷的错误创建函数
func NewConfigNotFoundError(configPath string, cause error) *XRFError {
	return &XRFError{
		Type:    ErrConfigNotFound,
		Message: fmt.Sprintf("配置文件或目录不存在: %s", configPath),
		Cause:   cause,
		Context: map[string]interface{}{
			"config_path": configPath,
		},
	}
}

func NewProtocolNotSupportedError(protocol string) *XRFError {
	return &XRFError{
		Type:    ErrProtocolNotSupported,
		Message: fmt.Sprintf("不支持的协议: %s", protocol),
		Context: map[string]interface{}{
			"protocol": protocol,
		},
	}
}

func NewPortInUseError(port int, cause error) *XRFError {
	return &XRFError{
		Type:    ErrPortInUse,
		Message: fmt.Sprintf("端口 %d 已被占用", port),
		Cause:   cause,
		Context: map[string]interface{}{
			"port": port,
		},
	}
}

func NewDomainRequiredError(protocol string) *XRFError {
	return &XRFError{
		Type:    ErrDomainRequired,
		Message: fmt.Sprintf("协议 %s 需要指定域名", protocol),
		Context: map[string]interface{}{
			"protocol": protocol,
		},
	}
}

func NewCertificateInvalidError(certPath string, cause error) *XRFError {
	return &XRFError{
		Type:    ErrCertificateInvalid,
		Message: fmt.Sprintf("证书文件无效: %s", certPath),
		Cause:   cause,
		Context: map[string]interface{}{
			"cert_path": certPath,
		},
	}
}

func NewSystemNotSupportedError(system, arch string) *XRFError {
	return &XRFError{
		Type:    ErrSystemNotSupported,
		Message: fmt.Sprintf("不支持的系统: %s %s", system, arch),
		Context: map[string]interface{}{
			"system":       system,
			"architecture": arch,
		},
	}
}

func NewPermissionDeniedError(path string, cause error) *XRFError {
	return &XRFError{
		Type:    ErrPermissionDenied,
		Message: fmt.Sprintf("权限不足: %s", path),
		Cause:   cause,
		Context: map[string]interface{}{
			"path": path,
		},
	}
}

func NewServiceNotRunningError(serviceName string) *XRFError {
	return &XRFError{
		Type:    ErrServiceNotRunning,
		Message: fmt.Sprintf("服务未运行: %s", serviceName),
		Context: map[string]interface{}{
			"service": serviceName,
		},
	}
}

func NewFileNotFoundError(filePath string, cause error) *XRFError {
	return &XRFError{
		Type:    ErrFileNotFound,
		Message: fmt.Sprintf("文件不存在: %s", filePath),
		Cause:   cause,
		Context: map[string]interface{}{
			"file_path": filePath,
		},
	}
}

func NewConfigInvalidError(reason string, cause error) *XRFError {
	return &XRFError{
		Type:    ErrConfigInvalid,
		Message: fmt.Sprintf("配置无效: %s", reason),
		Cause:   cause,
		Context: map[string]interface{}{
			"reason": reason,
		},
	}
}

// 辅助函数：检查错误类型
func IsXRFError(err error) bool {
	_, ok := err.(*XRFError)
	return ok
}

func GetXRFError(err error) *XRFError {
	if xrfErr, ok := err.(*XRFError); ok {
		return xrfErr
	}
	return nil
}

// 错误处理助手函数
func HandleError(err error) {
	if err == nil {
		return
	}

	if xrfErr := GetXRFError(err); xrfErr != nil {
		Error(xrfErr.GetFormattedError())
	} else {
		Error("发生错误: %v", err)
	}
}

// 警告处理函数
func HandleWarning(message string, suggestions ...string) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠️  %s\n", message))

	if len(suggestions) > 0 {
		sb.WriteString("\n💡 建议:\n")
		for i, suggestion := range suggestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
		}
	}

	Warning(sb.String())
}
