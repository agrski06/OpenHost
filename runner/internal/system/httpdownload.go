package system

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"
)

var httpClient = &http.Client{Timeout: 5 * time.Minute}

// Download fetches the given URL and writes it to destPath.
func Download(url, destPath string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", destPath, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}

	return nil
}

// DownloadWithRetry calls Download with exponential backoff retries.
func DownloadWithRetry(url, destPath string, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		lastErr = Download(url, destPath)
		if lastErr == nil {
			return nil
		}
		if attempt < maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
		}
	}
	return fmt.Errorf("download %s failed after %d retries: %w", url, maxRetries, lastErr)
}
