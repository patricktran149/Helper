package liquid

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	helper "github.com/patricktran149/Helper"
	"github.com/patricktran149/Helper/chatGPT"
	"github.com/patricktran149/liquid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"math/rand"
	"strings"
	"time"
)

var (
	chatGPTKey      string
	mgoTenantClient = make(map[string]*mongo.Client)
)

func NewEngine(openAIAPIKey string, tenantID string, mgoClient *mongo.Client) *liquid.Engine {
	engine := liquid.NewEngine()
	engine.RegisterFilter("right", rightNCharactersFilter)
	engine.RegisterFilter("left", leftNCharactersFilter)
	engine.RegisterFilter("substring", substringFilter)
	engine.RegisterFilter("raw", rawstringFilter)
	engine.RegisterFilter("randomInt", randomInt)
	engine.RegisterFilter("randomString", randomString)
	engine.RegisterFilter("chatGPT", chatGPTFilter)
	engine.RegisterFilter("dbLookup", dbLookup)
	engine.RegisterFilter("aesEncode", aesEncode)
	engine.RegisterFilter("aesDecode", aesDecode)

	if openAIAPIKey != "" && chatGPTKey != openAIAPIKey {
		chatGPTKey = openAIAPIKey
	}

	_, ok := mgoTenantClient[tenantID]
	if !ok && mgoClient != nil {
		mgoTenantClient[tenantID] = mgoClient
	}

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

// rightNCharactersFilter is a custom Liquid filter to extract the right n characters from a string.
func chatGPTFilter(input interface{}, request string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", errors.New("Input is not a string ")
	}

	request = fmt.Sprintf("%s : %s", request, str)

	response, err := chatGPT.Request(chatGPTKey, request)
	if err != nil {
		return "", errors.New("Request ChatGPT ERROR - " + err.Error())
	}

	return response, nil
}

func dbLookup(input interface{}, tenantID, tableName, fieldName string) (arrStr string, err error) {
	var array []bson.M

	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("Input is not a string ")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	opts := options.Find().SetLimit(10)

	cursor, err := mgoTenantClient[tenantID].Database(tenantID).Collection(tableName).Find(ctx, bson.M{fieldName: helper.RegexStringLike(str)}, opts)
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &array); err != nil {
		err = errors.New("Cursor All ERROR - " + err.Error())
		return
	}

	if err = cursor.Err(); err != nil {
		err = errors.New("Cursor.Err() ERROR - " + err.Error())
		return
	}

	arrStr = helper.JSONToString(array)

	return
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
