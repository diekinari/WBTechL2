package findAnagramms

import (
	"reflect"
	"sort"
	"testing"
)

// Helper: сравнивает два среза строк независимо от порядка
func equalStringSlicesIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	return reflect.DeepEqual(aa, bb)
}

func TestFindAnagramsBasic(t *testing.T) {
	input := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}
	wantGroups := map[string][]string{
		// ключи — те, которые создаст функция при первом вхождении (оригинальный первый элемент)
		"пятак":  {"пятка", "пятак", "тяпка"}, // порядок в срезе может отличаться
		"листок": {"слиток", "листок", "столик"},
	}

	got := FindAnagrams(input...)

	// Ожидаем ровно 2 группы
	if len(got) != len(wantGroups) {
		t.Fatalf("expected %d groups, got %d: %+v", len(wantGroups), len(got), got)
	}

	// Для каждой ожидаемой группы проверяем, что в got есть соответствующая группа
	for wantKey, wantSlice := range wantGroups {
		gotSlice, ok := got[wantKey]
		if !ok {
			t.Fatalf("expected group with key %q not found in result, result keys: %v", wantKey, keysOfMap(got))
		}
		if !equalStringSlicesIgnoreOrder(gotSlice, wantSlice) {
			t.Fatalf("group %q: expected items (any order) %v, got %v", wantKey, wantSlice, gotSlice)
		}
	}
}

func TestFindAnagramsEmptyAndSingleChars(t *testing.T) {
	// пустой ввод
	got := FindAnagrams()
	if len(got) != 0 {
		t.Fatalf("expected empty map for no input, got %v", got)
	}

	// только односимвольные строки — должны быть проигнорированы
	got = FindAnagrams("a", "b", "c", "д", "е")
	if len(got) != 0 {
		t.Fatalf("expected empty map for single-char inputs, got %v", got)
	}
}

func TestFindAnagramsWithDuplicates(t *testing.T) {
	// проверяем, что дубликаты попадают в группу
	input := []string{"abc", "bca", "abc", "cab", "xyz"}
	got := FindAnagrams(input...)

	// У нас должна быть одна группа (для "abc"):
	if len(got) != 1 {
		t.Fatalf("expected 1 group, got %d: %v", len(got), got)
	}

	// Найдём ключ (он должен быть "abc" — первый встретившийся)
	var key string
	for k := range got {
		key = k
		break
	}
	expected := []string{"bca", "abc", "abc", "cab"}

	if !equalStringSlicesIgnoreOrder(got[key], expected) {
		t.Fatalf("for key %q expected items (any order) %v, got %v", key, expected, got[key])
	}
}

// helper: список ключей map в срез
func keysOfMap(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
