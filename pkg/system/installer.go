package system

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/xrf-go/pkg/utils"
)

const (
	XrayReleasesAPI = "https://api.github.com/repos/XTLS/Xray-core/releases/latest"
	XrayBinaryPath  = "/usr/local/bin/xray"
	XrayConfigDir   = "/etc/xray"
	XrayConfsDir    = "/etc/xray/confs"
	GeositeURL      = "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat"
	GeoipURL        = "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat"
)

// GitHubRelease GitHub 发布信息
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Installer Xray 安装器
type Installer struct {
	detector *Detector
	verbose  bool
}

// NewInstaller 创建安装器
func NewInstaller(detector *Detector) *Installer {
	return &Installer{
		detector: detector,
		verbose:  false,
	}
}

// SetVerbose 设置详细输出
func (i *Installer) SetVerbose(verbose bool) {
	i.verbose = verbose
}

// InstallXray 安装 Xray
func (i *Installer) InstallXray() error {
	utils.PrintInfo("开始安装 Xray...")

	// 检查系统支持
	if supported, reason := i.detector.IsSupported(); !supported {
		return fmt.Errorf("系统不支持: %s", reason)
	}

	// 检查权限
	if !i.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能安装 Xray")
	}

	// 检查依赖
	missing := i.detector.CheckDependencies()
	if len(missing) > 0 {
		utils.PrintWarning("缺少依赖: %s", strings.Join(missing, ", "))
		if err := i.installDependencies(); err != nil {
			return fmt.Errorf("安装依赖失败: %w", err)
		}
	}

	// 获取最新版本信息
	release, err := i.getLatestRelease()
	if err != nil {
		return fmt.Errorf("获取最新版本信息失败: %w", err)
	}

	utils.PrintInfo("最新版本: %s", release.TagName)

	// 下载 Xray
	if err := i.downloadXray(release); err != nil {
		return fmt.Errorf("下载 Xray 失败: %w", err)
	}

	// 创建目录结构
	if err := i.createDirectories(); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 下载地理数据文件
	if err := i.downloadGeoFiles(); err != nil {
		utils.PrintWarning("下载地理数据文件失败: %v", err)
		utils.PrintInfo("您可以稍后手动下载这些文件")
	}

	utils.PrintSuccess("Xray 安装完成!")
	utils.PrintInfo("二进制文件: %s", XrayBinaryPath)
	utils.PrintInfo("配置目录: %s", XrayConfsDir)

	return nil
}

// getLatestRelease 获取最新版本信息
func (i *Installer) getLatestRelease() (*GitHubRelease, error) {
	if i.verbose {
		utils.PrintInfo("获取最新版本信息...")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(XrayReleasesAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// downloadXray 下载 Xray 二进制文件
func (i *Installer) downloadXray(release *GitHubRelease) error {
	// 获取适合当前系统的文件名
	binaryName, err := i.detector.GetXrayBinaryName()
	if err != nil {
		return err
	}

	// 查找对应的资源
	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, binaryName) && strings.HasSuffix(asset.Name, ".zip") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("未找到适合的二进制文件: %s", binaryName)
	}

	utils.PrintInfo("下载 Xray: %s", filepath.Base(downloadURL))

	// 下载到临时文件
	tempFile, err := i.downloadFile(downloadURL)
	if err != nil {
		return err
	}
	defer os.Remove(tempFile)

	// 解压并安装
	return i.extractAndInstall(tempFile)
}

// downloadFile 下载文件到临时位置
func (i *Installer) downloadFile(url string) (string, error) {
	client := &http.Client{Timeout: 300 * time.Second} // 5分钟超时
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "xray-*.zip")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 显示下载进度
	size := resp.ContentLength
	if size > 0 && i.verbose {
		utils.PrintInfo("文件大小: %.2f MB", float64(size)/1024/1024)
	}

	// 复制数据
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// extractAndInstall 解压并安装 Xray
func (i *Installer) extractAndInstall(zipFile string) error {
	if i.verbose {
		utils.PrintInfo("解压 Xray...")
	}

	// 打开ZIP文件
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// 查找 xray 可执行文件
	var xrayFile *zip.File
	for _, f := range r.File {
		if f.Name == "xray" || f.Name == "xray.exe" {
			xrayFile = f
			break
		}
	}

	if xrayFile == nil {
		return fmt.Errorf("在压缩包中未找到 xray 可执行文件")
	}

	// 提取 xray 文件
	rc, err := xrayFile.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// 创建目标文件
	destFile, err := os.OpenFile(XrayBinaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	// 复制文件内容
	_, err = io.Copy(destFile, rc)
	if err != nil {
		return err
	}

	if i.verbose {
		utils.PrintSuccess("Xray 二进制文件已安装到: %s", XrayBinaryPath)
	}

	return nil
}

// createDirectories 创建必要的目录结构
func (i *Installer) createDirectories() error {
	directories := []string{
		XrayConfigDir,
		XrayConfsDir,
		"/var/log/xray",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		if i.verbose {
			utils.PrintInfo("创建目录: %s", dir)
		}
	}

	return nil
}

// downloadGeoFiles 下载地理数据文件
func (i *Installer) downloadGeoFiles() error {
	files := map[string]string{
		"geosite.dat": GeositeURL,
		"geoip.dat":   GeoipURL,
	}

	for filename, url := range files {
		destPath := filepath.Join(XrayConfigDir, filename)
		if err := i.downloadGeoFile(url, destPath); err != nil {
			utils.PrintWarning("下载 %s 失败: %v", filename, err)
		} else if i.verbose {
			utils.PrintInfo("下载完成: %s", filename)
		}
	}

	return nil
}

// downloadGeoFile 下载单个地理数据文件
func (i *Installer) downloadGeoFile(url, destPath string) error {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	return err
}

// installDependencies 安装系统依赖
func (i *Installer) installDependencies() error {
	utils.PrintInfo("安装系统依赖...")

	packages := []string{"curl", "tar"}
	cmd, err := i.detector.GetInstallCommand(packages)
	if err != nil {
		return err
	}

	utils.PrintInfo("执行命令: %s", cmd)

	// 这里应该执行系统命令，但为了安全，我们只是提示用户
	utils.PrintWarning("请手动执行以下命令安装依赖:")
	utils.PrintInfo("  %s", cmd)

	return fmt.Errorf("请安装依赖后重新运行")
}

// IsInstalled 检查 Xray 是否已安装
func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(XrayBinaryPath)
	return err == nil
}

// GetInstalledVersion 获取已安装的版本
func (i *Installer) GetInstalledVersion() (string, error) {
	if !i.IsInstalled() {
		return "", fmt.Errorf("Xray 未安装")
	}

	// 执行 xray version 命令获取版本
	output, err := utils.ExecuteCommand(XrayBinaryPath, "version")
	if err != nil {
		return "", err
	}

	// 解析版本号
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Xray") && strings.Contains(line, "v") {
			// 提取版本号 (简单实现)
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "v") {
					return part, nil
				}
			}
		}
	}

	return "unknown", nil
}

// Uninstall 卸载 Xray
func (i *Installer) Uninstall() error {
	if !i.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能卸载 Xray")
	}

	utils.PrintInfo("卸载 Xray...")

	// 删除二进制文件
	if err := os.Remove(XrayBinaryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除二进制文件失败: %w", err)
	}

	// 询问是否删除配置文件
	utils.PrintWarning("配置目录 %s 包含您的配置，建议备份后手动删除", XrayConfigDir)

	utils.PrintSuccess("Xray 已卸载")
	return nil
}

// CheckUpdate 检查是否有更新
func (i *Installer) CheckUpdate() (bool, string, error) {
	currentVersion, err := i.GetInstalledVersion()
	if err != nil {
		return false, "", err
	}

	release, err := i.getLatestRelease()
	if err != nil {
		return false, "", err
	}

	latestVersion := release.TagName

	// 简单的版本比较
	if currentVersion != latestVersion {
		return true, latestVersion, nil
	}

	return false, latestVersion, nil
}

// Update 更新 Xray
func (i *Installer) Update() error {
	hasUpdate, latestVersion, err := i.CheckUpdate()
	if err != nil {
		return err
	}

	if !hasUpdate {
		utils.PrintInfo("Xray 已是最新版本")
		return nil
	}

	utils.PrintInfo("发现新版本: %s", latestVersion)
	utils.PrintInfo("开始更新...")

	// 备份当前版本
	backupPath := XrayBinaryPath + ".backup"
	if err := i.copyFile(XrayBinaryPath, backupPath); err != nil {
		utils.PrintWarning("备份当前版本失败: %v", err)
	}

	// 执行安装 (会覆盖现有文件)
	if err := i.InstallXray(); err != nil {
		// 恢复备份
		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, XrayBinaryPath)
		}
		return fmt.Errorf("更新失败: %w", err)
	}

	// 清理备份文件
	os.Remove(backupPath)

	utils.PrintSuccess("Xray 更新完成!")
	return nil
}

// copyFile 复制文件
func (i *Installer) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 复制权限
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// UpdateXray 更新到指定版本的 Xray
func (i *Installer) UpdateXray(version string) error {
	utils.PrintInfo("更新 Xray 到版本: %s", version)

	// 备份当前版本
	backupPath := XrayBinaryPath + ".backup"
	if i.IsInstalled() {
		if err := i.copyFile(XrayBinaryPath, backupPath); err != nil {
			utils.PrintWarning("备份当前版本失败: %v", err)
		}
	}

	// 执行安装 (会覆盖现有文件)
	if err := i.InstallXray(); err != nil {
		// 恢复备份
		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, XrayBinaryPath)
		}
		return fmt.Errorf("更新失败: %w", err)
	}

	// 清理备份文件
	os.Remove(backupPath)

	utils.PrintSuccess("Xray 更新完成!")
	return nil
}
