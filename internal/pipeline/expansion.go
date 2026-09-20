package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// coordinates semantic concept explosion over HTTP
type QueryExpander struct {
	OllamaURL string
	ModelName string
	Client    *http.Client
}

// configures secure and isolated local client
func NewQueryExpander() *QueryExpander {
	return &QueryExpander{
		OllamaURL: "http://localhost:11434/api/generate",
		ModelName: "llama3.2:1b",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// extracts 3 to 5 optimized subqueries
func (qe *QueryExpander) FanOut(userQuery string) ([]string, error) {
	prompt := fmt.Sprintf(`You are the first stage of an advanced AI search engine. 
Take this conversational query, strip out filler words, and extract 3 to 5 unique, highly specific, technical keyword search queries targeting the raw source files.

User Query: "%s"

Return ONLY a raw JSON array of strings. Do not include markdown formatting, backticks, or conversational text.
Example Output Format: ["term 1", "term 2", "term 3"]`, userQuery)

	// build strict structured request payload
	requestPayload := map[string]interface{}{
		"model":  qe.ModelName,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1, // try to force deterministic + non-creative keyword extraction
		},
	}

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// ship payload to local model with isolated handshake
	resp, err := qe.Client.Post(qe.OllamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama connection failed: %w", err)
	}
	defer resp.Body.Close()

	var ollamaResponse struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response text: %w", err)
	}

	// clean formatting if model ignores instructions
	cleanJSON := strings.TrimSpace(ollamaResponse.Response)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	// decode JSON text directly into typed Go slice
	var subQueries []string
	if err := json.Unmarshal([]byte(cleanJSON), &subQueries); err != nil {
		// fallback: split raw string tokens so pipeline won't break
		return []string{userQuery}, nil
	}

	return subQueries, nil
}
