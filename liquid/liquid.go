package liquid

import (
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"

	"github.com/patricktran149/liquid"
)

func NewEngine() *liquid.Engine {
	engine := liquid.NewEngine()
	engine.RegisterFilter("right", rightNCharactersFilter)
	engine.RegisterFilter("left", leftNCharactersFilter)
	engine.RegisterFilter("substring", substringFilter)
	engine.RegisterFilter("raw", rawstringFilter)
	engine.RegisterFilter("randomInt", randomInt)
	engine.RegisterFilter("randomString", randomString)
	engine.RegisterFilter("aesEncode", aesEncode)
	engine.RegisterFilter("aesDecode", aesDecode)
	engine.RegisterFilter("subtractLeft", subtractLeft)
	engine.RegisterFilter("subtractRight", subtractRight)

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

func randomInt(input interface{}) (int, error) {
	length, ok := input.(int)
	if !ok {
		return 0, fmt.Errorf("Input is not a number ")
	}

	if length <= 0 {
		return 0, fmt.Errorf("Input is invalid: negative number ")
	}

	// Calculate the range for the random number based on the number of length
	min := intPow(10, length-1)
	max := intPow(10, length) - 1

	// Generate a random number within the specified range
	return rand.Intn(max-min+1) + min, nil
}

func randomString(input interface{}) (string, error) {
	length, ok := input.(int)
	if !ok {
		return "", fmt.Errorf("Input is not a number ")
	}

	// Define the range of printable ASCII characters
	const printableChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Generate a random string by selecting characters from the printableChars
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = printableChars[rand.Intn(len(printableChars))]
	}

	return string(result), nil
}

func intPow(base, exponent int) int {
	result := 1
	for i := 0; i < exponent; i++ {
		result *= base
	}
	return result
}

func aesEncode(input interface{}, key string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", errors.New("Input is not a string ")
	}

	if len(key) != 32 {
		return "", errors.New("Key length is not 32 ")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", errors.New("AES New Cipher ERROR - " + err.Error())
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("Cipher New GCM ERROR - " + err.Error())
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(cryptoRand.Reader, nonce); err != nil {
		return "", errors.New("Read Nonce ERROR - " + err.Error())
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(str), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func aesDecode(input interface{}, key string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", errors.New("Input is not a string ")
	}

	if len(key) != 32 {
		return "", errors.New("Key length is not 32 ")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", errors.New("AES New Cipher ERROR - " + err.Error())
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("Cipher New GCM ERROR - " + err.Error())
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("Cipher text too short ")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("AESGCM open Nonce ERROR - " + err.Error())
	}

	return string(plaintext), nil
}

// Remove everything after the last splitCharacter including the splitCharacter
func subtractRight(input interface{}, splitChar string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	if str == "" {
		return "", nil
	}

	lastDashIndex := strings.LastIndex(str, splitChar)
	if lastDashIndex == -1 {
		return str, nil
	}

	return str[:lastDashIndex], nil
}

// Remove everything before the first splitCharacter including the splitCharacter
func subtractLeft(input interface{}, splitChar string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	if str == "" {
		return "", nil
	}

	firstDashIndex := strings.Index(str, splitChar)
	if firstDashIndex == -1 {
		return str, nil
	}

	return str[firstDashIndex+1:], nil
}
