package util

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lithammer/dedent"
)

func SprintfDedent(format string, a ...any) string {
	val := format
	val = strings.Replace(val, "\t", "  ", -1)
	val = dedent.Dedent(val)
	val = fmt.Sprintf(val, a...)
	val = strings.TrimSpace(val)
	return val
}

func Join(sep string, vals ...string) string {
	return strings.Join(vals, sep)
}

func RemoveDupOf(s string, target rune) string {
	var buf bytes.Buffer
	var last rune
	for i, r := range s {
		if (i == 0 || r != target) ||
			(r == target && r != last) {

			buf.WriteRune(r)
			last = r
		}
	}
	return buf.String()
}

func SanitizeNameForKubernetes(name string) (string, error) {
	newName := strings.Map(func(r rune) rune {
		switch {
		case '0' <= r && r <= '9':
			fallthrough
		case 'A' <= r && r <= 'Z':
			fallthrough
		case 'a' <= r && r <= 'z':
			return r
		default:
			return '-'
		}
	}, name)

	newName = strings.Trim(newName, "-")
	newName = RemoveDupOf(newName, '-')

	return newName, CheckNameLengthForKubernetes(newName)
}

func CheckNameLengthForKubernetes(name string) error {
	if len(name) > 63 {
		return fmt.Errorf("name length exceeds 63 characters: %s (%d characters)", name, len(name))
	}

	return nil
}
