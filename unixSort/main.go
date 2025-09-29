package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// SortFlags хранит значения флагов
type SortFlags struct {
	Column   int
	Numeric  bool
	Reverse  bool
	Unique   bool
	Month    bool
	IgnoreWS bool
	Check    bool
	HumanNum bool
	File     string
}

func parseFlags() *SortFlags {
	flags := &SortFlags{}

	pflag.IntVarP(&flags.Column, "key", "k", 0, "Номер колонки для сортировки (1-based). По умолчанию вся строка")
	pflag.BoolVarP(&flags.Numeric, "numeric", "n", false, "Сортировать как числа")
	pflag.BoolVarP(&flags.Reverse, "reverse", "r", false, "Обратный порядок")
	pflag.BoolVarP(&flags.Unique, "unique", "u", false, "Выводить только уникальные строки")
	pflag.BoolVarP(&flags.Month, "month", "M", false, "Сортировать по названию месяца (Jan..Dec)")
	pflag.BoolVarP(&flags.IgnoreWS, "ignore-trailing-space", "b", false, "Игнорировать хвостовые пробелы")
	pflag.BoolVarP(&flags.Check, "check", "c", false, "Проверить: отсортированы ли данные")
	pflag.BoolVarP(&flags.HumanNum, "human-numeric-sort", "h", false, "Числовая сортировка с суффиксами (1K, 2M...)")

	pflag.Parse()
	if args := pflag.Args(); len(args) > 0 {
		flags.File = args[0]
	}
	return flags
}

// readLines читает все строки из файла или stdin.
// Увеличиваем буфер Scanner для длинных строк.
func readLines(file string) (ls []string, fErr error) {
	var scanner *bufio.Scanner
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				fErr = errors.Join(fErr, err)
			}
		}(f)
		scanner = bufio.NewScanner(f)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	// увеличить максимальный размер токена (до 10 MiB)
	const maxTok = 10 * 1024 * 1024
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, maxTok)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// getKey возвращает ключ для сортировки — колонку N (1-based) или всю строку.
// Разделитель столбцов — табуляция.
func getKey(line string, col int) string {
	if col <= 0 {
		return line
	}
	cols := strings.Split(line, "\t")
	if col <= len(cols) {
		return cols[col-1]
	}
	return "" // если колонки нет, пустая строка
}

// parseNumeric пытается распарсить строку в float64
func parseNumeric(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// поддерживаем запятую как десятичный? будем требовать точку для простоты
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, true
	}
	return 0, false
}

// parseHuman читаем числа с суффиксами K, M, G, T (десятичные: 1K = 1000)
func parseHuman(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// отделим суффикс
	n := len(s)
	if n == 0 {
		return 0, false
	}
	// возможно последний символ - буква (K/M/G/T)
	last := s[n-1]
	mult := 1.0
	body := s
	if (last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') {
		body = strings.TrimSpace(s[:n-1])
		switch strings.ToUpper(string(last)) {
		case "K":
			mult = 1e3
		case "M":
			mult = 1e6
		case "G":
			mult = 1e9
		case "T":
			mult = 1e12
		default:
			// неизвестный суффикс — пробуем парсить всю строку как число
			body = s
			mult = 1.0
		}
	}
	if v, err := strconv.ParseFloat(body, 64); err == nil {
		return v * mult, true
	}
	return 0, false
}

// monthValue: Jan -> 1, Feb -> 2, ... Dec -> 12
func monthValue(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	// берем первые 3 буквы
	if len(s) >= 3 {
		s = s[:3]
	}
	switch strings.ToLower(s) {
	case "jan":
		return 1, true
	case "feb":
		return 2, true
	case "mar":
		return 3, true
	case "apr":
		return 4, true
	case "may":
		return 5, true
	case "jun":
		return 6, true
	case "jul":
		return 7, true
	case "aug":
		return 8, true
	case "sep":
		return 9, true
	case "oct":
		return 10, true
	case "nov":
		return 11, true
	case "dec":
		return 12, true
	default:
		return 0, false
	}
}

// compareKeys сравнивает два ключа согласно флагам. Возвращает -1 если a<b, 0 если =, +1 если a>b.
func compareKeys(a, b string, flags *SortFlags) int {
	// применим -b (игнорировать хвостовые пробелы)
	if flags.IgnoreWS {
		a = strings.TrimRight(a, " \t")
		b = strings.TrimRight(b, " \t")
	}

	// приоритет: Month -> HumanNum (-h) -> Numeric (-n) -> лексикографически
	if flags.Month {
		ma, oka := monthValue(a)
		mb, okb := monthValue(b)
		if oka && okb {
			if ma < mb {
				return -1
			} else if ma > mb {
				return 1
			}
			return 0
		}
		// если хоть одна не распозналась, падение к строковому сравнению
	}

	if flags.HumanNum {
		va, oka := parseHuman(a)
		vb, okb := parseHuman(b)
		if oka && okb {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
		// если не удалось распарсить суффиксы — падение дальше
	}

	if flags.Numeric {
		va, oka := parseNumeric(a)
		vb, okb := parseNumeric(b)
		if oka && okb {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
		// если не распарсилось в числа — падение к строковому сравнению
	}

	// лексикографическое сравнение (по байтам/Unicode)
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// sortLines сортирует слайс строк по флагам
func sortLines(lines []string, flags *SortFlags) []string {
	// по флагу -b можно предварительно убрать trailing spaces из всей строки,
	// чтобы поведение уник/проверки/печати было предсказуемым
	if flags.IgnoreWS {
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " \t")
		}
	}

	less := func(i, j int) bool {
		var aKey, bKey string
		aKey = getKey(lines[i], flags.Column)
		bKey = getKey(lines[j], flags.Column)

		cmp := compareKeys(aKey, bKey, flags)
		if cmp < 0 {
			return true
		} else if cmp > 0 {
			return false
		}
		// если ключи равны — сохранить стабильность: сравниваем полные строки для детерминированности
		if lines[i] < lines[j] {
			return true
		}
		return false
	}

	// сортируем
	sort.Slice(lines, func(i, j int) bool {
		res := less(i, j)
		if flags.Reverse {
			return !res
		}
		return res
	})

	// -u: убрать дубликаты
	if flags.Unique {
		lines = unique(lines)
	}

	return lines
}

// unique: удаляет подряд идущие одинаковые строки (после сортировки)
func unique(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	prev := lines[0]
	out = append(out, prev)
	for i := 1; i < len(lines); i++ {
		if lines[i] == prev {
			continue
		}
		prev = lines[i]
		out = append(out, prev)
	}
	return out
}

// checkSorted проверяет отсортированность без полного выделения памяти/без изменения оригинала.
// Возвращает true, если отсортировано.
func checkSorted(lines []string, flags *SortFlags) bool {
	if len(lines) < 2 {
		return true
	}
	comparePair := func(a, b string) int {
		aKey := getKey(a, flags.Column)
		bKey := getKey(b, flags.Column)
		return compareKeys(aKey, bKey, flags)
	}
	for i := 1; i < len(lines); i++ {
		cmp := comparePair(lines[i-1], lines[i])
		// cmp < 0 => prev < cur
		// cmp == 0 => equal, then compare full line
		if cmp < 0 {
			if flags.Reverse {
				// expected descending; but found ascending pair => not sorted
				return false
			}
			continue
		} else if cmp > 0 {
			// prev > cur
			if flags.Reverse {
				// descending order: prev > cur is ok
				continue
			}
			return false
		} else {
			// keys equal -> compare whole strings to be deterministic
			if lines[i-1] == lines[i] {
				continue
			}
			// if not equal, decide with full string comparison
			if lines[i-1] < lines[i] {
				if flags.Reverse {
					return false
				}
				continue
			} else if lines[i-1] > lines[i] {
				if flags.Reverse {
					continue
				}
				return false
			}
		}
	}
	return true
}

func main() {
	flags := parseFlags()

	lines, err := readLines(flags.File)
	if err != nil {
		_, err := fmt.Fprintln(os.Stderr, "Ошибка чтения:", err)
		if err != nil {
			return
		}
		os.Exit(2)
	}

	if flags.Check {
		ok := checkSorted(lines, flags)
		if ok {
			fmt.Println("Файл отсортирован")
			os.Exit(0)
		} else {
			fmt.Println("Файл НЕ отсортирован")
			os.Exit(1)
		}
	}

	result := sortLines(lines, flags)

	// Выводим отсортированные строки
	w := bufio.NewWriter(os.Stdout)
	for _, line := range result {
		_, err := fmt.Fprintln(w, line)
		if err != nil {
			return
		}
	}
	err = w.Flush()
	if err != nil {
		_, err = fmt.Fprintln(os.Stderr, err)
		return
	}
}
