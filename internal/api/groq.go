package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Choice struct {
	Message Message `json:"message"`
}
type ResponseBody struct {
	Choices []Choice `json:"choices"`
}

// GetGroqChatCompletion is a generic function to interact with the Groq Chat API.
// It takes the model name and a slice of messages as parameters.
func GetGroqChatCompletion(apiKey, modelName string, messages []Message) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Groq API key was not provided")
	}
	if modelName == "" {
		return "", fmt.Errorf("model name was not provided")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("at least one message is required")
	}

	// Build the request body from the provided parameters.
	requestData := RequestBody{
		Model:    modelName,
		Messages: messages,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("error encoding JSON: %w", err)
	}

	// --- API Call Logic ---
	url := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"API returned a non-success status: %d, %s",
			resp.StatusCode,
			string(body),
		)
	}

	// --- Parse the response ---
	var responseBody ResponseBody
	if err := json.Unmarshal(body, &responseBody); err != nil {
		return "", fmt.Errorf("error decoding response JSON: %w", err)
	}

	if len(responseBody.Choices) > 0 && responseBody.Choices[0].Message.Content != "" {
		// Return the content from the first choice's message.
		return responseBody.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("API response did not contain a valid choice")
}
