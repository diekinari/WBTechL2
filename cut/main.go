package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseFields возвращает множество нужных полей (от 1 до N), например: 1,2,4,5
func parseFields(fields string) map[int]struct{} {
	out := make(map[int]struct{})
	if fields == "" {
		return out
	}
	parts := strings.Split(fields, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			r := strings.SplitN(part, "-", 2)
			start, _ := strconv.Atoi(r[0])
			end, _ := strconv.Atoi(r[1])
			for i := start; i <= end; i++ {
				if i > 0 {
					out[i] = struct{}{}
				}
			}
		} else {
			i, _ := strconv.Atoi(part)
			if i > 0 {
				out[i] = struct{}{}
			}
		}
	}
	return out
}

func main() {
	fFields := flag.String("f", "", "fields")
	delim := flag.String("d", "\t", "delimiter (default TAB)")
	onlySep := flag.Bool("s", false, "only lines with delimiter")
	flag.Parse()

	fields := parseFields(*fFields)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if *onlySep && !strings.Contains(line, *delim) {
			continue
		}
		cols := strings.Split(line, *delim)
		var res []string
		for i := 1; i <= len(cols); i++ {
			if _, ok := fields[i]; ok {
				res = append(res, cols[i-1])
			}
		}
		if len(res) > 0 {
			fmt.Println(strings.Join(res, *delim))
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "read error:", err)
		os.Exit(1)
	}
}