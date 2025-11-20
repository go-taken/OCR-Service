package pkg

import "strings"

func RemoveExtraSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
