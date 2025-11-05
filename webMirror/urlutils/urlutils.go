package urlutils

import (
	"net/url"
	"path/filepath"
	"strings"
)

// NormalizeURL нормализует URL, убирая фрагменты и параметры при необходимости
func NormalizeURL(rawURL string, baseURL *url.URL) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Если относительный URL, разрешаем относительно baseURL
	if u.Scheme == "" {
		u = baseURL.ResolveReference(u)
	}

	// Убираем фрагмент (#anchor)
	u.Fragment = ""

	// Нормализуем путь
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}

	return u, nil
}

// IsSameDomain проверяет, принадлежит ли URL тому же домену
func IsSameDomain(targetURL, baseURL *url.URL) bool {
	return targetURL.Scheme == baseURL.Scheme && targetURL.Host == baseURL.Host
}

// URLToLocalPath преобразует URL в локальный путь файла
func URLToLocalPath(u *url.URL, basePath string) string {
	// Создаем путь из домена и пути URL
	path := u.Path
	if path == "/" || path == "" {
		path = "/index.html"
	}

	// Убираем начальный слеш
	path = strings.TrimPrefix(path, "/")

	// Если путь заканчивается на /, добавляем index.html
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}

	// Если путь не имеет расширения и похож на директорию, добавляем .html
	if !strings.Contains(filepath.Base(path), ".") {
		path = path + ".html"
	}

	// Создаем полный путь
	fullPath := filepath.Join(basePath, u.Host, path)

	// Очищаем путь от недопустимых символов
	fullPath = filepath.Clean(fullPath)

	return fullPath
}

// URLToResourcePath преобразует URL ресурса в локальный путь
func URLToResourcePath(u *url.URL, basePath string) string {
	path := strings.TrimPrefix(u.Path, "/")

	// Если путь пустой, используем имя файла из пути
	if path == "" {
		path = "resource"
	}

	fullPath := filepath.Join(basePath, u.Host, path)
	return filepath.Clean(fullPath)
}

// GetResourceType определяет тип ресурса по URL
func GetResourceType(u *url.URL) string {
	path := strings.ToLower(u.Path)
	ext := filepath.Ext(path)

	switch ext {
	case ".css":
		return "css"
	case ".js":
		return "js"
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".ico":
		return "image"
	case ".html", ".htm":
		return "html"
	case ".pdf", ".zip", ".tar", ".gz":
		return "document"
	default:
		// Проверяем по Content-Type если есть
		return "unknown"
	}
}

// IsResourceURL проверяет, является ли URL ресурсом (CSS, JS, изображение и т.д.)
func IsResourceURL(u *url.URL) bool {
	resourceType := GetResourceType(u)
	return resourceType == "css" || resourceType == "js" || resourceType == "image" || resourceType == "document"
}

// LocalPathToURL преобразует локальный путь обратно в относительный URL для HTML
func LocalPathToURL(filePath, basePath string, baseURL *url.URL) string {
	// Получаем относительный путь от basePath
	relPath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return ""
	}

	// Заменяем обратные слеши на прямые (для Windows)
	relPath = filepath.ToSlash(relPath)

	// Если путь начинается с .., значит он вне basePath
	if strings.HasPrefix(relPath, "../") {
		return ""
	}

	// Добавляем начальный слеш
	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}

	return relPath
}
