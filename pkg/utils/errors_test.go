package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestXRFError(t *testing.T) {
	t.Run("BasicErrorCreation", func(t *testing.T) {
		err := NewProtocolNotSupportedError("invalid-protocol")

		if err.Type != ErrProtocolNotSupported {
			t.Errorf("Expected error type %d, got %d", ErrProtocolNotSupported, err.Type)
		}

		if !strings.Contains(err.Message, "invalid-protocol") {
			t.Errorf("Error message should contain protocol name")
		}

		if err.Context["protocol"] != "invalid-protocol" {
			t.Errorf("Context should contain protocol name")
		}
	})

	t.Run("ErrorWithCause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewConfigNotFoundError("/path/to/config", cause)

		if err.Cause != cause {
			t.Errorf("Error cause not properly set")
		}

		if err.Unwrap() != cause {
			t.Errorf("Unwrap should return the cause")
		}
	})

	t.Run("GetSuggestions", func(t *testing.T) {
		err := NewPortInUseError(8080, nil)
		suggestions := err.GetSuggestions()

		if len(suggestions) == 0 {
			t.Error("Should have suggestions for port in use error")
		}

		found := false
		for _, suggestion := range suggestions {
			if strings.Contains(suggestion, "check-port") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should suggest checking port availability")
		}
	})

	t.Run("FormattedError", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewPortInUseError(8080, cause)

		formatted := err.GetFormattedError()

		// 应该包含错误图标
		if !strings.Contains(formatted, "❌") {
			t.Error("Formatted error should contain error icon")
		}

		// 应该包含原始错误信息
		if !strings.Contains(formatted, "connection refused") {
			t.Error("Formatted error should contain cause")
		}

		// 应该包含建议图标
		if !strings.Contains(formatted, "💡") {
			t.Error("Formatted error should contain suggestion icon")
		}

		// 应该包含详细信息
		if !strings.Contains(formatted, "📋") {
			t.Error("Formatted error should contain context icon")
		}
	})
}

func TestErrorTypes(t *testing.T) {
	testCases := []struct {
		name        string
		createError func() *XRFError
		errorType   ErrorType
		expectField string
	}{
		{
			name:        "ConfigNotFound",
			createError: func() *XRFError { return NewConfigNotFoundError("/test/config", nil) },
			errorType:   ErrConfigNotFound,
			expectField: "config_path",
		},
		{
			name:        "ProtocolNotSupported",
			createError: func() *XRFError { return NewProtocolNotSupportedError("test-protocol") },
			errorType:   ErrProtocolNotSupported,
			expectField: "protocol",
		},
		{
			name:        "PortInUse",
			createError: func() *XRFError { return NewPortInUseError(8080, nil) },
			errorType:   ErrPortInUse,
			expectField: "port",
		},
		{
			name:        "DomainRequired",
			createError: func() *XRFError { return NewDomainRequiredError("vless-reality") },
			errorType:   ErrDomainRequired,
			expectField: "protocol",
		},
		{
			name:        "CertificateInvalid",
			createError: func() *XRFError { return NewCertificateInvalidError("/cert/path", nil) },
			errorType:   ErrCertificateInvalid,
			expectField: "cert_path",
		},
		{
			name:        "SystemNotSupported",
			createError: func() *XRFError { return NewSystemNotSupportedError("windows", "amd64") },
			errorType:   ErrSystemNotSupported,
			expectField: "system",
		},
		{
			name:        "PermissionDenied",
			createError: func() *XRFError { return NewPermissionDeniedError("/etc/xray", nil) },
			errorType:   ErrPermissionDenied,
			expectField: "path",
		},
		{
			name:        "ServiceNotRunning",
			createError: func() *XRFError { return NewServiceNotRunningError("xray") },
			errorType:   ErrServiceNotRunning,
			expectField: "service",
		},
		{
			name:        "FileNotFound",
			createError: func() *XRFError { return NewFileNotFoundError("/missing/file", nil) },
			errorType:   ErrFileNotFound,
			expectField: "file_path",
		},
		{
			name:        "ConfigInvalid",
			createError: func() *XRFError { return NewConfigInvalidError("syntax error", nil) },
			errorType:   ErrConfigInvalid,
			expectField: "reason",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.createError()

			// 检查错误类型
			if err.Type != tc.errorType {
				t.Errorf("Expected error type %d, got %d", tc.errorType, err.Type)
			}

			// 检查上下文字段
			if _, exists := err.Context[tc.expectField]; !exists {
				t.Errorf("Expected context field %s", tc.expectField)
			}

			// 检查建议
			suggestions := err.GetSuggestions()
			if len(suggestions) == 0 {
				t.Errorf("Error type %s should have suggestions", tc.name)
			}

			// 检查格式化输出
			formatted := err.GetFormattedError()
			if len(formatted) == 0 {
				t.Errorf("Formatted error should not be empty")
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsXRFError", func(t *testing.T) {
		// 测试 XRF 错误
		xrfErr := NewProtocolNotSupportedError("test")
		if !IsXRFError(xrfErr) {
			t.Error("IsXRFError should return true for XRF error")
		}

		// 测试普通错误
		normalErr := errors.New("normal error")
		if IsXRFError(normalErr) {
			t.Error("IsXRFError should return false for normal error")
		}

		// 测试 nil
		if IsXRFError(nil) {
			t.Error("IsXRFError should return false for nil")
		}
	})

	t.Run("GetXRFError", func(t *testing.T) {
		// 测试 XRF 错误
		originalErr := NewPortInUseError(8080, nil)
		retrievedErr := GetXRFError(originalErr)
		if retrievedErr == nil {
			t.Error("GetXRFError should return XRF error")
		}
		if retrievedErr != originalErr {
			t.Error("Retrieved error should be the same instance")
		}

		// 测试普通错误
		normalErr := errors.New("normal error")
		if GetXRFError(normalErr) != nil {
			t.Error("GetXRFError should return nil for normal error")
		}

		// 测试 nil
		if GetXRFError(nil) != nil {
			t.Error("GetXRFError should return nil for nil")
		}
	})
}

func TestSuggestionQuality(t *testing.T) {
	testCases := []struct {
		errorType        ErrorType
		expectedKeywords []string
	}{
		{
			errorType:        ErrPortInUse,
			expectedKeywords: []string{"check-port", "端口", "netstat"},
		},
		{
			errorType:        ErrProtocolNotSupported,
			expectedKeywords: []string{"list", "协议", "别名"},
		},
		{
			errorType:        ErrConfigNotFound,
			expectedKeywords: []string{"init", "权限", "安装"},
		},
		{
			errorType:        ErrServiceNotRunning,
			expectedKeywords: []string{"start", "status", "logs"},
		},
		{
			errorType:        ErrPermissionDenied,
			expectedKeywords: []string{"sudo", "权限"},
		},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.errorType)), func(t *testing.T) {
			err := &XRFError{Type: tc.errorType}
			suggestions := err.GetSuggestions()

			if len(suggestions) == 0 {
				t.Errorf("Error type %d should have suggestions", tc.errorType)
				return
			}

			suggestionText := strings.Join(suggestions, " ")

			for _, keyword := range tc.expectedKeywords {
				if !strings.Contains(suggestionText, keyword) {
					t.Errorf("Suggestions for error type %d should contain keyword '%s'", tc.errorType, keyword)
				}
			}
		})
	}
}

func TestErrorContextHandling(t *testing.T) {
	t.Run("ContextInFormattedError", func(t *testing.T) {
		err := &XRFError{
			Type:    ErrPortInUse,
			Message: "Test error",
			Context: map[string]interface{}{
				"port":    8080,
				"service": "test-service",
				"pid":     1234,
			},
		}

		formatted := err.GetFormattedError()

		// 检查所有上下文字段是否出现在格式化输出中
		contextFields := []string{"port", "service", "pid"}
		for _, field := range contextFields {
			if !strings.Contains(formatted, field) {
				t.Errorf("Formatted error should contain context field: %s", field)
			}
		}

		// 检查上下文值
		if !strings.Contains(formatted, "8080") {
			t.Error("Formatted error should contain port value")
		}
		if !strings.Contains(formatted, "test-service") {
			t.Error("Formatted error should contain service name")
		}
		if !strings.Contains(formatted, "1234") {
			t.Error("Formatted error should contain PID")
		}
	})

	t.Run("EmptyContextHandling", func(t *testing.T) {
		err := &XRFError{
			Type:    ErrConfigInvalid,
			Message: "Test error",
			Context: map[string]interface{}{},
		}

		formatted := err.GetFormattedError()

		// 空上下文时不应该显示详细信息部分
		if strings.Contains(formatted, "📋 详细信息:") {
			t.Error("Formatted error should not show context section when context is empty")
		}
	})
}
