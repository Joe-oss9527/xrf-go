package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (h *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func (h *HTTPClient) Download(url, outputPath string) error {
	resp, err := h.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (h *HTTPClient) DownloadWithProgress(url, outputPath string, progressFn func(downloaded, total int64)) error {
	resp, err := h.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	var downloaded int64
	total := resp.ContentLength

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write file: %w", writeErr)
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, total)
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read response: %w", err)
		}
	}

	return nil
}

func (h *HTTPClient) Head(url string) (*http.Response, error) {
	resp, err := h.client.Head(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make HEAD request: %w", err)
	}

	return resp, nil
}

func (h *HTTPClient) CheckURL(url string) error {
	resp, err := h.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("URL check failed with status: %s", resp.Status)
	}

	return nil
}

func GetPublicIP() (string, error) {
	client := NewHTTPClient(10 * time.Second)

	urls := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ipinfo.io/ip",
		"https://checkip.amazonaws.com",
	}

	for _, url := range urls {
		if ip, err := client.Get(url); err == nil {
			return string(ip), nil
		}
	}

	return "", fmt.Errorf("failed to get public IP from all sources")
}

func TestHTTPConnectivity() error {
	client := NewHTTPClient(5 * time.Second)

	testURLs := []string{
		"https://www.google.com",
		"https://www.cloudflare.com",
		"https://github.com",
	}

	for _, url := range testURLs {
		if err := client.CheckURL(url); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no internet connectivity detected")
}

func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func CalculateDownloadSpeed(bytes int64, duration time.Duration) string {
	if duration == 0 {
		return "0 B/s"
	}
	bytesPerSecond := float64(bytes) / duration.Seconds()
	return FormatBytes(int64(bytesPerSecond)) + "/s"
}

var DefaultHTTPClient = NewHTTPClient(30 * time.Second)
