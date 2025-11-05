package downloader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader управляет загрузкой ресурсов
type Downloader struct {
	client      *http.Client
	userAgent   string
	timeout     time.Duration
	concurrency int
	semaphore   chan struct{}
}

// NewDownloader создает новый загрузчик
func NewDownloader(timeout time.Duration, concurrency int) *Downloader {
	d := &Downloader{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Следуем редиректам, но ограничиваем их количество
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		userAgent:   "WebMirror/1.0",
		timeout:     timeout,
		concurrency: concurrency,
		semaphore:   make(chan struct{}, concurrency),
	}
	return d
}

// DownloadFile загружает файл по URL и сохраняет его локально
func (d *Downloader) DownloadFile(targetURL *url.URL, localPath string) error {
	// Получаем семафор для ограничения параллельности
	d.semaphore <- struct{}{}
	defer func() { <-d.semaphore }()

	// Создаем директорию если нужно
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "*/*")

	// Выполняем запрос
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", targetURL.String(), err)
	}
	defer resp.Body.Close()

	// Проверяем статус код
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, targetURL.String())
	}

	// Создаем файл
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", localPath, err)
	}
	defer file.Close()

	// Копируем содержимое
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", localPath, err)
	}

	return nil
}

// GetContentType получает Content-Type ресурса
func (d *Downloader) GetContentType(targetURL *url.URL) (string, error) {
	req, err := http.NewRequest("HEAD", targetURL.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return "application/octet-stream", nil
	}

	// Извлекаем только MIME тип (без параметров)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}

	return strings.TrimSpace(contentType), nil
}

// FetchContent загружает содержимое ресурса в память
func (d *Downloader) FetchContent(targetURL *url.URL) ([]byte, string, error) {
	d.semaphore <- struct{}{}
	defer func() { <-d.semaphore }()

	req, err := http.NewRequest("GET", targetURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch %s: %w", targetURL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, targetURL.String())
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read content: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	return content, contentType, nil
}

// CheckRobotsTxt проверяет robots.txt для URL
func (d *Downloader) CheckRobotsTxt(baseURL *url.URL) (bool, error) {
	robotsURL := *baseURL
	robotsURL.Path = "/robots.txt"

	req, err := http.NewRequest("GET", robotsURL.String(), nil)
	if err != nil {
		return true, nil // Если не можем проверить, разрешаем
	}

	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return true, nil // Если не можем загрузить, разрешаем
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return true, nil // Если robots.txt не существует, разрешаем
	}

	// Простой парсер robots.txt
	// В реальной реализации нужно использовать библиотеку для парсинга
	// Для упрощения разрешаем все
	return true, nil
}

