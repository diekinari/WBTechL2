package mirror

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"WBTechL2/webMirror/cssparser"
	"WBTechL2/webMirror/downloader"
	"WBTechL2/webMirror/htmlparser"
	"WBTechL2/webMirror/urlutils"
)

// Mirror управляет процессом зеркалирования сайта
type Mirror struct {
	baseURL        *url.URL
	basePath       string
	maxDepth       int
	downloader     *downloader.Downloader
	visitedURLs    map[string]bool
	urlToLocalPath map[string]string
	htmlFiles      map[string]string // URL -> local path для HTML файлов
	cssFiles       map[string]string // URL -> local path для CSS файлов
	mu             sync.RWMutex
	wg             sync.WaitGroup
	errors         []error
	errMu          sync.Mutex
}

// NewMirror создает новый экземпляр зеркалирования
func NewMirror(startURL string, outputPath string, maxDepth int, timeout int, concurrency int) (*Mirror, error) {
	baseURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Нормализуем URL
	if baseURL.Scheme == "" {
		baseURL.Scheme = "http"
	}
	if baseURL.Path == "" {
		baseURL.Path = "/"
	}

	// Создаем директорию для вывода
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	m := &Mirror{
		baseURL:        baseURL,
		basePath:       outputPath,
		maxDepth:       maxDepth,
		downloader:     downloader.NewDownloader(time.Duration(timeout)*time.Second, concurrency),
		visitedURLs:    make(map[string]bool),
		urlToLocalPath: make(map[string]string),
		htmlFiles:      make(map[string]string),
		cssFiles:       make(map[string]string),
		errors:         make([]error, 0),
	}

	return m, nil
}

// Start начинает процесс зеркалирования
func (m *Mirror) Start() error {
	fmt.Printf("Starting mirror of %s\n", m.baseURL.String())
	fmt.Printf("Output directory: %s\n", m.basePath)
	fmt.Printf("Max depth: %d\n", m.maxDepth)

	// Начинаем с корневого URL
	m.wg.Add(1)
	go m.processURL(m.baseURL, 0, "")

	m.wg.Wait()

	// Финальный проход: обновляем все HTML и CSS файлы с правильными ссылками
	fmt.Println("\nUpdating links in HTML and CSS files...")
	m.updateAllLinks()

	// Выводим ошибки если есть
	if len(m.errors) > 0 {
		fmt.Printf("\nCompleted with %d errors:\n", len(m.errors))
		for _, err := range m.errors {
			fmt.Printf("  - %v\n", err)
		}
	}

	return nil
}

// processURL обрабатывает один URL
func (m *Mirror) processURL(targetURL *url.URL, depth int, referrer string) {
	defer m.wg.Done()

	// Проверяем глубину
	if depth > m.maxDepth {
		return
	}

	// Нормализуем URL
	normalizedURL, err := urlutils.NormalizeURL(targetURL.String(), m.baseURL)
	if err != nil {
		m.addError(fmt.Errorf("failed to normalize URL %s: %w", targetURL.String(), err))
		return
	}

	// Проверяем, не посещали ли мы этот URL
	m.mu.RLock()
	normalizedStr := normalizedURL.String()
	if m.visitedURLs[normalizedStr] {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	// Проверяем, тот же ли это домен
	if !urlutils.IsSameDomain(normalizedURL, m.baseURL) {
		return
	}

	// Отмечаем как посещенный
	m.mu.Lock()
	m.visitedURLs[normalizedStr] = true
	m.mu.Unlock()

	fmt.Printf("[%d] Downloading: %s\n", depth, normalizedURL.String())

	// Загружаем содержимое
	content, contentType, err := m.downloader.FetchContent(normalizedURL)
	if err != nil {
		m.addError(fmt.Errorf("failed to download %s: %w", normalizedURL.String(), err))
		return
	}

	// Определяем локальный путь
	var localPath string
	if urlutils.IsResourceURL(normalizedURL) || !isHTMLContent(contentType) {
		// Это ресурс (CSS, JS, изображение)
		localPath = urlutils.URLToResourcePath(normalizedURL, m.basePath)
	} else {
		// Это HTML страница
		localPath = urlutils.URLToLocalPath(normalizedURL, m.basePath)
	}

	// Сохраняем файл
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		m.addError(fmt.Errorf("failed to create directory for %s: %w", localPath, err))
		return
	}

	if err := os.WriteFile(localPath, content, 0644); err != nil {
		m.addError(fmt.Errorf("failed to save file %s: %w", localPath, err))
		return
	}

	// Сохраняем маппинг URL -> локальный путь
	relativePath := urlutils.LocalPathToURL(localPath, m.basePath, m.baseURL)
	m.mu.Lock()
	m.urlToLocalPath[normalizedStr] = relativePath
	m.mu.Unlock()

	// Если это HTML, парсим и обрабатываем ссылки
	if isHTMLContent(contentType) {
		// Парсим HTML для извлечения ссылок
		links, err := htmlparser.ExtractLinks(strings.NewReader(string(content)), m.baseURL)
		if err != nil {
			m.addError(fmt.Errorf("failed to parse HTML from %s: %w", normalizedURL.String(), err))
			return
		}

		// Создаем маппинг для замены ссылок
		urlMap := make(map[string]string)
		for _, link := range links {
			linkURL, err := urlutils.NormalizeURL(link.URL, normalizedURL)
			if err != nil {
				continue
			}

			// Проверяем, тот же ли домен
			if !urlutils.IsSameDomain(linkURL, m.baseURL) {
				continue
			}

			linkStr := linkURL.String()
			m.mu.RLock()
			if localPath, ok := m.urlToLocalPath[linkStr]; ok {
				// Ссылка уже скачана, используем локальный путь
				urlMap[link.URL] = localPath
			}
			m.mu.RUnlock()

			// Добавляем в очередь для скачивания
			m.mu.RLock()
			visited := m.visitedURLs[linkStr]
			m.mu.RUnlock()

			if !visited {
				m.wg.Add(1)
				go m.processURL(linkURL, depth+1, normalizedStr)
			}
		}

		// Заменяем ссылки в HTML на локальные пути
		// Используем полный маппинг для всех ссылок
		m.mu.RLock()
		fullURLMap := make(map[string]string)
		for _, link := range links {
			linkURL, err := urlutils.NormalizeURL(link.URL, normalizedURL)
			if err == nil && urlutils.IsSameDomain(linkURL, m.baseURL) {
				linkStr := linkURL.String()
				if localPath, ok := m.urlToLocalPath[linkStr]; ok {
					// Используем оригинальный URL из ссылки для маппинга
					fullURLMap[link.URL] = localPath
				}
			}
		}
		m.mu.RUnlock()

		// Объединяем с уже имеющимся маппингом
		for k, v := range urlMap {
			fullURLMap[k] = v
		}

		updatedHTML := htmlparser.ReplaceLinks(string(content), fullURLMap)

		// Сохраняем обновленный HTML (временно, затем обновим в финальном проходе)
		if err := os.WriteFile(localPath, []byte(updatedHTML), 0644); err != nil {
			m.addError(fmt.Errorf("failed to save updated HTML %s: %w", localPath, err))
		}

		// Сохраняем информацию о HTML файле для финального обновления
		m.mu.Lock()
		m.htmlFiles[normalizedStr] = localPath
		m.mu.Unlock()
	} else if isCSSContent(contentType) {
		// Если это CSS файл, извлекаем и обрабатываем ссылки
		cssLinks, err := cssparser.ExtractCSSLinks(strings.NewReader(string(content)), normalizedURL)
		if err != nil {
			m.addError(fmt.Errorf("failed to parse CSS from %s: %w", normalizedURL.String(), err))
			return
		}

		// Создаем маппинг для замены ссылок в CSS
		cssURLMap := make(map[string]string)
		for _, cssLink := range cssLinks {
			linkURL, err := urlutils.NormalizeURL(cssLink, normalizedURL)
			if err != nil {
				continue
			}

			if !urlutils.IsSameDomain(linkURL, m.baseURL) {
				continue
			}

			linkStr := linkURL.String()
			m.mu.RLock()
			if localPath, ok := m.urlToLocalPath[linkStr]; ok {
				cssURLMap[cssLink] = localPath
			}
			m.mu.RUnlock()

			// Добавляем в очередь для скачивания
			m.mu.RLock()
			visited := m.visitedURLs[linkStr]
			m.mu.RUnlock()

			if !visited {
				m.wg.Add(1)
				go m.processURL(linkURL, depth+1, normalizedStr)
			}
		}

		// Заменяем ссылки в CSS
		updatedCSS := cssparser.ReplaceCSSLinks(string(content), cssURLMap)

		// Сохраняем обновленный CSS (временно, затем обновим в финальном проходе)
		if err := os.WriteFile(localPath, []byte(updatedCSS), 0644); err != nil {
			m.addError(fmt.Errorf("failed to save updated CSS %s: %w", localPath, err))
		}

		// Сохраняем информацию о CSS файле для финального обновления
		m.mu.Lock()
		m.cssFiles[normalizedStr] = localPath
		m.mu.Unlock()
	}
}

// addError добавляет ошибку в список
func (m *Mirror) addError(err error) {
	m.errMu.Lock()
	defer m.errMu.Unlock()
	m.errors = append(m.errors, err)
}

// isHTMLContent проверяет, является ли содержимое HTML
func isHTMLContent(contentType string) bool {
	return contentType == "text/html" || contentType == "application/xhtml+xml"
}

// isCSSContent проверяет, является ли содержимое CSS
func isCSSContent(contentType string) bool {
	return contentType == "text/css" || contentType == "text/css; charset=utf-8"
}

// updateAllLinks обновляет все ссылки в HTML и CSS файлах после завершения загрузки
func (m *Mirror) updateAllLinks() {
	m.mu.RLock()
	htmlFiles := make(map[string]string)
	cssFiles := make(map[string]string)
	urlToLocalPath := make(map[string]string)
	for k, v := range m.htmlFiles {
		htmlFiles[k] = v
	}
	for k, v := range m.cssFiles {
		cssFiles[k] = v
	}
	for k, v := range m.urlToLocalPath {
		urlToLocalPath[k] = v
	}
	m.mu.RUnlock()

	// Обновляем HTML файлы
	for urlStr, localPath := range htmlFiles {
		url, err := url.Parse(urlStr)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(localPath)
		if err != nil {
			m.addError(fmt.Errorf("failed to read HTML file %s: %w", localPath, err))
			continue
		}

		// Извлекаем все ссылки из HTML
		links, err := htmlparser.ExtractLinks(strings.NewReader(string(content)), m.baseURL)
		if err != nil {
			continue
		}

		// Создаем полный маппинг для замены
		urlMap := make(map[string]string)
		for _, link := range links {
			linkURL, err := urlutils.NormalizeURL(link.URL, url)
			if err != nil {
				continue
			}

			if !urlutils.IsSameDomain(linkURL, m.baseURL) {
				continue
			}

			linkStr := linkURL.String()
			if localPath, ok := urlToLocalPath[linkStr]; ok {
				urlMap[link.URL] = localPath
			}
		}

		// Заменяем ссылки
		updatedHTML := htmlparser.ReplaceLinks(string(content), urlMap)

		// Сохраняем обновленный HTML
		if err := os.WriteFile(localPath, []byte(updatedHTML), 0644); err != nil {
			m.addError(fmt.Errorf("failed to update HTML file %s: %w", localPath, err))
		}
	}

	// Обновляем CSS файлы
	for urlStr, localPath := range cssFiles {
		url, err := url.Parse(urlStr)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(localPath)
		if err != nil {
			m.addError(fmt.Errorf("failed to read CSS file %s: %w", localPath, err))
			continue
		}

		// Извлекаем все ссылки из CSS
		cssLinks, err := cssparser.ExtractCSSLinks(strings.NewReader(string(content)), url)
		if err != nil {
			continue
		}

		// Создаем маппинг для замены
		cssURLMap := make(map[string]string)
		for _, cssLink := range cssLinks {
			linkURL, err := urlutils.NormalizeURL(cssLink, url)
			if err != nil {
				continue
			}

			if !urlutils.IsSameDomain(linkURL, m.baseURL) {
				continue
			}

			linkStr := linkURL.String()
			if localPath, ok := urlToLocalPath[linkStr]; ok {
				cssURLMap[cssLink] = localPath
			}
		}

		// Заменяем ссылки
		updatedCSS := cssparser.ReplaceCSSLinks(string(content), cssURLMap)

		// Сохраняем обновленный CSS
		if err := os.WriteFile(localPath, []byte(updatedCSS), 0644); err != nil {
			m.addError(fmt.Errorf("failed to update CSS file %s: %w", localPath, err))
		}
	}
}
