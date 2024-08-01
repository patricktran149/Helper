package chatGPT

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func Request(key, request string) (response string, err error) {
	apiURL := "https://api.openai.com/v1/chat/completions"
	// Define the request payload
	requestBody := OpenAIRequest{
		Model: "gpt-4",
		Messages: []Message{
			{
				Role:    "user",
				Content: request,
			},
		},
		Temperature: 0,
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
		break
	}

	return response, nil
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Error   struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Choice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}
