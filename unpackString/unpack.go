package unpackString

import (
	"fmt"
	"strconv"
)

func unpackString(s string) (string, error) {
	if len(s) == 0 {
		return s, nil
	}
	sRunes := []rune(s)
	result := make([]rune, 0, len(sRunes))
	for i := 0; i < len(sRunes); i++ {
		curr := sRunes[i]
		// Если текущая руна – это число:
		if num, err := strconv.Atoi(string(curr)); err == nil {
			// Проверяем на корректность предыдущего ввода
			if len(result) == 0 {
				return "", fmt.Errorf("invalid input(no letters before digits)")
			}
			repeatedRune := result[len(result)-1]
			// Экранирование
			if repeatedRune == '\\' {
				result[len(result)-1] = curr
				continue
			}
			// Обычная распаковка числа как множителя строки
			for j := 0; j < (num - 1); j++ {
				result = append(result, repeatedRune)
			}
			// Если руна – не число, то добавляем в буфер
		} else {
			result = append(result, curr)
		}
	}
	return string(result), nil
}
