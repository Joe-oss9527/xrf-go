package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreConfig_BackupFileCleanup(t *testing.T) {
	// 创建测试目录
	testDir := "/tmp/xrf-restore-cleanup-test"
	os.RemoveAll(testDir)
	defer os.RemoveAll(testDir)

	// 初始化配置管理器
	os.Setenv("XRF_TEST_MODE", "1")
	defer os.Unsetenv("XRF_TEST_MODE")

	configMgr := NewConfigManager(testDir)
	if err := configMgr.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 添加一个测试协议
	params := map[string]interface{}{
		"port":   10001,
		"uuid":   "test-uuid",
		"domain": "test.example.com",
		"path":   "/test",
		"email":  "test@example.com",
	}

	if err := configMgr.AddProtocol("vw", "test-vless", params); err != nil {
		t.Fatalf("AddProtocol failed: %v", err)
	}

	// 创建备份
	backupPath := "/tmp/test-backup-for-restore.tar.gz"
	defer os.Remove(backupPath)

	if err := configMgr.BackupConfig(backupPath); err != nil {
		t.Fatalf("BackupConfig failed: %v", err)
	}

	// 记录恢复前当前目录的备份文件数量
	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir("/tmp"); err != nil {
		t.Fatalf("Failed to change directory to /tmp: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWorkDir); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	beforeFiles, _ := filepath.Glob("xrf-current-backup-*.tar.gz")
	beforeCount := len(beforeFiles)

	// 执行恢复
	if err := configMgr.RestoreConfig(backupPath); err != nil {
		t.Fatalf("RestoreConfig failed: %v", err)
	}

	// 检查恢复后的备份文件
	afterFiles, _ := filepath.Glob("xrf-current-backup-*.tar.gz")
	afterCount := len(afterFiles)

	// 清理测试生成的备份文件
	for _, f := range afterFiles {
		os.Remove(f)
	}

	// 验证结果
	if afterCount > beforeCount {
		t.Errorf("RestoreConfig created %d new backup file(s) that were not cleaned up", afterCount-beforeCount)
		t.Errorf("Before: %d files, After: %d files", beforeCount, afterCount)
		if len(afterFiles) > 0 {
			t.Errorf("New backup files created: %v", afterFiles[beforeCount:])
		}
	}
}
