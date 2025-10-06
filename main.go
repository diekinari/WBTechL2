package main

import (
	"WBTechL2/findAnagramms"
	"fmt"
)

func main() {
	entry := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}
	res := findAnagramms.FindAnagrams(entry...)
	fmt.Println(res)
}
