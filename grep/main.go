package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"regexp"
	"strings"
)

type sortFlags struct {
	AfterLines       int  // A
	BeforeLines      int  // B
	BothLines        int  // C
	ShowCount        bool // c
	IgnoreCase       bool // i
	Invert           bool // v
	TreatAsString    bool // F
	ShowLinesNumbers bool // n
}

func parseFlags() *sortFlags {
	flags := &sortFlags{}

	pflag.IntVarP(&flags.AfterLines, "A", "A", 0, "после каждой найденной строки дополнительно вывести N строк после неё (контекст).")
	pflag.IntVarP(&flags.BeforeLines, "B", "B", 0, "вывести N строк до каждой найденной строки.")
	pflag.IntVarP(&flags.BothLines, "C", "C", 0, "вывести N строк контекста вокруг найденной строки (включает и до, и после; эквивалентно -A N -B N)/")
	pflag.BoolVarP(&flags.ShowCount, "c", "c", false, "выводить только то количество строк, что совпадающих с шаблоном (т.е. вместо самих строк — число).")
	pflag.BoolVarP(&flags.IgnoreCase, "i", "i", false, "игнорировать регистр.")
	pflag.BoolVarP(&flags.Invert, "v", "v", false, "инвертировать фильтр: выводить строки, не содержащие шаблон.")
	pflag.BoolVarP(&flags.TreatAsString, "F", "F", false, "воспринимать шаблон как фиксированную строку, а не регулярное выражение (т.е. выполнять точное совпадение подстроки).")
	pflag.BoolVarP(&flags.ShowLinesNumbers, "n", "n", false, " выводить номер строки перед каждой найденной строкой.")
	return flags
}

func match(line, pattern string, flags *sortFlags, regex *regexp.Regexp) bool {
	matched := false
	if flags.TreatAsString {
		if flags.IgnoreCase {
			matched = strings.Contains(strings.ToLower(line), strings.ToLower(pattern))
		} else {
			matched = strings.Contains(line, pattern)
		}
	} else {
		if flags.IgnoreCase {
			matched = regex.MatchString(line)
		} else {
			matched = regex.MatchString(line)
		}
	}
	if flags.Invert {
		return !matched
	}
	return matched
}

func main() {
	flags := parseFlags()
	pflag.Parse()

	// Проверяем наличие шаблона для поиска
	args := pflag.Args()
	if len(args) < 1 {
		panic("Usage: grep [OPTIONS] PATTERN [FILE]")
	}
	pattern := args[0]

	var file *os.File
	var err error
	if len(args) > 1 && args[1] != "-" {
		file, err = os.Open(args[1])
		if err != nil {
			panic(err)
		}
		defer file.Close()
	} else {
		file = os.Stdin
	}

	// BothLines (-C) перекрывает -A и -B
	if flags.BothLines > 0 {
		flags.AfterLines = flags.BothLines
		flags.BeforeLines = flags.BothLines
	}

	var regex *regexp.Regexp
	if !flags.TreatAsString {
		var pat = pattern
		if flags.IgnoreCase {
			pat = "(?i)" + pattern
		}
		regex, err = regexp.Compile(pat)
		if err != nil {
			panic("Invalid pattern: " + err.Error())
		}
	}

	// Прочитаем все строки файла в память
	scanner := bufio.NewScanner(file)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	matchIndexes := make([]int, 0)
	for i, line := range allLines {
		if match(line, pattern, flags, regex) {
			matchIndexes = append(matchIndexes, i)
		}
	}

	if flags.ShowCount {
		fmt.Println(len(matchIndexes))
		return
	}

	// Набор диапазонов с контекстом
	type interval struct{ start, end int }
	intervals := make([]interval, 0)
	for _, idx := range matchIndexes {
		st := idx - flags.BeforeLines
		if st < 0 {
			st = 0
		}
		en := idx + flags.AfterLines
		if en > len(allLines)-1 {
			en = len(allLines) - 1
		}
		intervals = append(intervals, interval{st, en})
	}
	// Объединяем пересекающиеся
	merged := make([]interval, 0)
	for _, iv := range intervals {
		if len(merged) == 0 {
			merged = append(merged, iv)
			continue
		}
		prev := &merged[len(merged)-1]
		if iv.start <= prev.end+1 {
			if iv.end > prev.end {
				prev.end = iv.end
			}
		} else {
			merged = append(merged, iv)
		}
	}

	// Для повторяющихся совпадений выводим "--" между разными областями (как в UNIX grep)
	printed := make(map[int]bool)
	for mi, iv := range merged {
		if mi > 0 {
			fmt.Println("--")
		}
		for i := iv.start; i <= iv.end; i++ {
			if printed[i] {
				continue // не повторять строку
			}
			printed[i] = true
			out := ""
			if flags.ShowLinesNumbers {
				out = fmt.Sprintf("%d:", i+1)
			}
			fmt.Println(out + allLines[i])
		}
	}
}
