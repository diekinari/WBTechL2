package findAnagramms

import (
	"sort"
)

func FindAnagrams(ss ...string) map[string][]string {
	set := make(map[string][]string)
	tempSet := make(map[string]string)

	for _, s := range ss {
		runes := []rune(s)
		if len(runes) <= 1 {
			continue
		}

		sortedRunes := make([]rune, len(runes), len(runes))
		copy(sortedRunes, runes)
		sort.Slice(sortedRunes, func(i, j int) bool {
			return sortedRunes[i] < sortedRunes[j]
		})

		sortedKey := string(sortedRunes)
		// if we've met the same anagramm before, we add it to the final set
		if originalKey, ok := tempSet[sortedKey]; ok {
			_, ok := set[originalKey]
			if ok {
				set[originalKey] = append(set[originalKey], s)
			} else {
				set[originalKey] = []string{s, originalKey}
			}
			continue
		} else {
			// tempSet haven't met this anagramm before, so it records it
			tempSet[sortedKey] = s
		}

	}
	return set
}
