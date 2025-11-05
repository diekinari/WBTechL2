package cssparser

import (
	"io"
	"net/url"
	"regexp"
	"strings"
)

// ExtractCSSLinks извлекает ссылки из CSS файла (@import и url())
func ExtractCSSLinks(r io.Reader, baseURL *url.URL) ([]string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var links []string
	cssContent := string(content)

	// Извлекаем @import
	importRegex := regexp.MustCompile(`@import\s+(?:url\()?["']?([^"')]+)["']?\)?`)
	importMatches := importRegex.FindAllStringSubmatch(cssContent, -1)
	for _, match := range importMatches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}

	// Извлекаем url()
	urlRegex := regexp.MustCompile(`url\s*\(\s*["']?([^"')]+)["']?\s*\)`)
	urlMatches := urlRegex.FindAllStringSubmatch(cssContent, -1)
	for _, match := range urlMatches {
		if len(match) > 1 {
			urlStr := match[1]
			// Пропускаем data: и другие специальные протоколы
			if !strings.HasPrefix(urlStr, "data:") && !strings.HasPrefix(urlStr, "javascript:") {
				links = append(links, urlStr)
			}
		}
	}

	return links, nil
}

// ReplaceCSSLinks заменяет ссылки в CSS на локальные пути
func ReplaceCSSLinks(cssContent string, urlMap map[string]string) string {
	// Заменяем @import
	importRegex := regexp.MustCompile(`@import\s+(?:url\()?["']?([^"')]+)["']?\)?`)
	cssContent = importRegex.ReplaceAllStringFunc(cssContent, func(match string) string {
		submatches := importRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			oldURL := submatches[1]
			if newPath, ok := urlMap[oldURL]; ok {
				return strings.Replace(match, oldURL, newPath, 1)
			}
		}
		return match
	})

	// Заменяем url()
	urlRegex := regexp.MustCompile(`url\s*\(\s*["']?([^"')]+)["']?\s*\)`)
	cssContent = urlRegex.ReplaceAllStringFunc(cssContent, func(match string) string {
		submatches := urlRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			oldURL := submatches[1]
			if newPath, ok := urlMap[oldURL]; ok {
				// Определяем, есть ли кавычки в оригинале
				if strings.Contains(match, `"`) {
					return `url("` + newPath + `")`
				} else if strings.Contains(match, `'`) {
					return `url('` + newPath + `')`
				} else {
					return `url(` + newPath + `)`
				}
			}
		}
		return match
	})

	return cssContent
}

