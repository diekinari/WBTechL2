package main

import (
	"flag"
	"fmt"
	"os"

	"WBTechL2/webMirror/mirror"
)

func main() {
	// Парсим флаги
	var (
		urlFlag     = flag.String("url", "", "URL сайта для зеркалирования (обязательный)")
		outputFlag  = flag.String("output", "./mirror", "Директория для сохранения зеркала")
		depthFlag   = flag.Int("depth", 3, "Максимальная глубина рекурсии (количество уровней ссылок)")
		timeoutFlag = flag.Int("timeout", 30, "Таймаут для HTTP запросов в секундах")
		concFlag    = flag.Int("concurrency", 5, "Количество одновременных загрузок")
		_           = flag.Bool("robots", false, "Проверять robots.txt (опционально) - функциональность пока не реализована")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Утилита для зеркалирования веб-сайтов, аналогичная wget -m\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПримеры:\n")
		fmt.Fprintf(os.Stderr, "  %s -url https://example.com -output ./example_mirror -depth 2\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -url http://localhost:8080 -depth 5 -concurrency 10\n", os.Args[0])
	}

	flag.Parse()

	// Проверяем обязательный параметр URL
	if *urlFlag == "" {
		fmt.Fprintf(os.Stderr, "Ошибка: необходимо указать URL с помощью флага -url\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Валидируем параметры
	if *depthFlag < 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: глубина должна быть >= 0\n")
		os.Exit(1)
	}

	if *timeoutFlag <= 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: таймаут должен быть > 0\n")
		os.Exit(1)
	}

	if *concFlag <= 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: количество одновременных загрузок должно быть > 0\n")
		os.Exit(1)
	}

	// Создаем экземпляр зеркалирования
	m, err := mirror.NewMirror(*urlFlag, *outputFlag, *depthFlag, *timeoutFlag, *concFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при создании зеркала: %v\n", err)
		os.Exit(1)
	}

	// Запускаем зеркалирование
	if err := m.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при зеркалировании: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nЗеркалирование завершено успешно!")
	fmt.Printf("Результаты сохранены в: %s\n", *outputFlag)
}
