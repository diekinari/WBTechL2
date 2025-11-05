package htmlparser

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ResourceLink представляет ссылку на ресурс
type ResourceLink struct {
	URL      string
	Type     string // "css", "js", "image", "link"
	AttrName string // имя атрибута (href, src, etc.)
}

// ExtractLinks извлекает все ссылки из HTML документа
func ExtractLinks(r io.Reader, baseURL *url.URL) ([]ResourceLink, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var links []ResourceLink

	// Функция для обхода узлов
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				// Ссылки на страницы
				href := getAttr(n, "href")
				if href != "" {
					links = append(links, ResourceLink{
						URL:      href,
						Type:     "link",
						AttrName: "href",
					})
				}
			case "link":
				// CSS файлы и другие ресурсы
				href := getAttr(n, "href")
				rel := getAttr(n, "rel")
				if href != "" {
					resourceType := "link"
					if rel == "stylesheet" || strings.HasSuffix(href, ".css") {
						resourceType = "css"
					}
					links = append(links, ResourceLink{
						URL:      href,
						Type:     resourceType,
						AttrName: "href",
					})
				}
			case "script":
				// JavaScript файлы
				src := getAttr(n, "src")
				if src != "" {
					links = append(links, ResourceLink{
						URL:      src,
						Type:     "js",
						AttrName: "src",
					})
				}
			case "img":
				// Изображения
				src := getAttr(n, "src")
				if src != "" {
					links = append(links, ResourceLink{
						URL:      src,
						Type:     "image",
						AttrName: "src",
					})
				}
				// Также проверяем srcset для изображений
				srcset := getAttr(n, "srcset")
				if srcset != "" {
					// Парсим srcset (может содержать несколько URL)
					urls := parseSrcset(srcset)
					for _, u := range urls {
						links = append(links, ResourceLink{
							URL:      u,
							Type:     "image",
							AttrName: "srcset",
						})
					}
				}
			case "source":
				// Источники для picture и audio/video
				src := getAttr(n, "src")
				if src != "" {
					links = append(links, ResourceLink{
						URL:      src,
						Type:     "image",
						AttrName: "src",
					})
				}
			case "video", "audio":
				// Видео и аудио файлы
				src := getAttr(n, "src")
				if src != "" {
					links = append(links, ResourceLink{
						URL:      src,
						Type:     "document",
						AttrName: "src",
					})
				}
			case "iframe", "embed", "object":
				// Встроенные ресурсы
				src := getAttr(n, "src")
				if src == "" {
					src = getAttr(n, "data")
				}
				if src != "" {
					links = append(links, ResourceLink{
						URL:      src,
						Type:     "link",
						AttrName: "src",
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return links, nil
}

// ReplaceLinks заменяет ссылки в HTML на локальные пути
func ReplaceLinks(htmlContent string, urlMap map[string]string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	// Функция для замены атрибутов
	var replaceAttrs func(*html.Node)
	replaceAttrs = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Заменяем href и src атрибуты
			for i, attr := range n.Attr {
				if attr.Key == "href" || attr.Key == "src" {
					if newPath, ok := urlMap[attr.Val]; ok {
						n.Attr[i].Val = newPath
					}
				}
				// Также обрабатываем srcset
				if attr.Key == "srcset" {
					newSrcset := replaceSrcset(attr.Val, urlMap)
					n.Attr[i].Val = newSrcset
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			replaceAttrs(c)
		}
	}

	replaceAttrs(doc)

	// Преобразуем обратно в строку
	var buf strings.Builder
	html.Render(&buf, doc)
	return buf.String()
}

// getAttr получает значение атрибута из узла
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// parseSrcset парсит атрибут srcset и извлекает URL
func parseSrcset(srcset string) []string {
	var urls []string
	parts := strings.Split(srcset, ",")
	for _, part := range parts {
		// Srcset может содержать URL и дескриптор (например, "image.jpg 2x")
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 {
			urls = append(urls, fields[0])
		}
	}
	return urls
}

// replaceSrcset заменяет URL в srcset
func replaceSrcset(srcset string, urlMap map[string]string) string {
	parts := strings.Split(srcset, ",")
	var newParts []string
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 {
			oldURL := fields[0]
			if newPath, ok := urlMap[oldURL]; ok {
				// Заменяем URL, сохраняя дескриптор
				if len(fields) > 1 {
					newParts = append(newParts, newPath+" "+strings.Join(fields[1:], " "))
				} else {
					newParts = append(newParts, newPath)
				}
			} else {
				newParts = append(newParts, part)
			}
		}
	}
	return strings.Join(newParts, ", ")
}
