package liquid

import (
	"fmt"
	"github.com/osteele/liquid"
	"strings"
)

func NewEngine() *liquid.Engine {
	engine := liquid.NewEngine()
	engine.RegisterFilter("right", rightNCharactersFilter)
	engine.RegisterFilter("left", leftNCharactersFilter)
	engine.RegisterFilter("substring", substringFilter)
	engine.RegisterFilter("raw", rawstringFilter)

	return engine
}

// rightNCharactersFilter is a custom Liquid filter to extract the right n characters from a string.
func rightNCharactersFilter(input interface{}, n int) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	if n >= len(str) {
		return str, nil // Return the full string if n is greater than or equal to its length
	}

	return str[len(str)-n:], nil
}

// leftNCharactersFilter is a custom Liquid filter to extract the left n characters from a string.
func leftNCharactersFilter(input interface{}, n int) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	if n >= len(str) {
		return str, nil // Return the full string if n is greater than or equal to its length
	}

	return str[:n], nil
}

// substringFilter is a custom Liquid filter to extract a substring from index n to index m from a string.
func substringFilter(input interface{}, n int, m int) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	if n >= len(str) || m < 0 || n > m {
		return "", fmt.Errorf("invalid index range")
	}

	if m >= len(str) {
		m = len(str) - 1
	}

	return str[n : m+1], nil
}

// substringFilter is a custom Liquid filter to extract a substring from index n to index m from a string.
func rawstringFilter(input interface{}) string {
	str, ok := input.(string)
	if !ok {
		return ""
	}

	replacer := strings.NewReplacer(
		`\`, `\\`, // Replace \ with \\
		`"`, `\"`, // Replace " with \"
		"\n", "\\n", // Replace newline with \n
		"\t", "\\t", // Replace tab with \t
		"\r", "\\r", // Replace carriage return with \r
		"\b", "\\b", // Replace backspace with \b
		"\f", "\\f", // Replace form feed with \f
		"\v", "\\v", // Replace vertical tab with \v
		"\a", "\\a", // Replace alert with \a
		"\x00", "\\x00", // Replace null character with \x00
		// Add more replacements for other special characters as needed
	)

	return replacer.Replace(str)
}
