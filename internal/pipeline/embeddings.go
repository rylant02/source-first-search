package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EmbeddingsEngine struct {
	OllamaURL string
	ModelName string
	Client    *http.Client
}

func NewEmbeddingsEngine() *EmbeddingsEngine {
	return &EmbeddingsEngine{
		OllamaURL: "http://localhost:11434/api/embeddings",
		ModelName: "all-minilm",
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// transforms str to 384-D vector slice (Ollama)
func (ee *EmbeddingsEngine) GetVector(text string) ([]float32, error) {
	payload := map[string]string{
		"model":  ee.ModelName,
		"prompt": text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	resp, err := qeClientPost(ee.OllamaURL, jsonData, ee.Client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding vector: %w", err)
	}

	return result.Embedding, nil
}

// keeps network requests clean
func qeClientPost(url string, data []byte, client *http.Client) (*http.Response, error) {
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("network connection to embedding engine failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("embedding engine returned non-200 status: %d", resp.StatusCode)
	}
	return resp, nil
}
