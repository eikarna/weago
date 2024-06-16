package functions

import "strings"

func RemoveColonDigits(input string) string {
	parts := strings.Split(input, "@")
	localPart := strings.Split(parts[0], ":")[0]
	return localPart + "@" + parts[1]
}
