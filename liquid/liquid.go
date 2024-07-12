package liquid

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/patricktran149/liquid"
	"io"
	"math/rand"
	"net/http"
	"strings"
)

var chatGPTKey string

func NewEngine(openAIAPIKey string) *liquid.Engine {
	engine := liquid.NewEngine()
	engine.RegisterFilter("right", rightNCharactersFilter)
	engine.RegisterFilter("left", leftNCharactersFilter)
	engine.RegisterFilter("substring", substringFilter)
	engine.RegisterFilter("raw", rawstringFilter)
	engine.RegisterFilter("randomInt", randomInt)
	engine.RegisterFilter("randomString", randomString)
	engine.RegisterFilter("chatGPT", chatGPT)

	if openAIAPIKey != "" {
		chatGPTKey = openAIAPIKey
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
func chatGPT(input interface{}, request string) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("input is not a string")
	}

	request = fmt.Sprintf("%s : %s", request, str)

	response, err := chatGPTRequest(chatGPTKey, request)
	if err != nil {
		return "", errors.New("Request ChatGPT ERROR - " + err.Error())
	}

	return response, nil
}

func chatGPTRequest(key, request string) (response string, err error) {
	apiURL := "https://api.openai.com/v1/chat/completions"
	// Define the request payload
	requestBody := OpenAIRequest{
		Model: "gpt-3.5-turbo-0125",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: request,
			},
		},
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", errors.New("JSON Marshal Request body ERROR - " + err.Error())
	}

	// Create and send the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return "", errors.New("Creating request ERROR - " + err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("OpenAI-Organization", orgID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("Making request ERROR - " + err.Error())
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("Reading response ERROR - " + err.Error())
	}

	var openAIResp OpenAIResponse
	err = json.Unmarshal(body, &openAIResp)
	if err != nil {
		return "", errors.New("Unmarshalling response ERROR - " + err.Error())
	}

	// Check for errors in the response
	if openAIResp.Error.Message != "" {
		return "", errors.New("API ERROR - " + openAIResp.Error.Message)
	}

	// Print the response
	for _, choice := range openAIResp.Choices {
		response = choice.Message.Content
	}

	return response, nil
}

type OpenAIRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
