package clientSide

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// MatchingSvc demonstrates how to use the HTTP client
type MatchingSvc struct {
	client HTTPClient
	url    string
}

// NewMatchingSvc creates a new matching service instance
func NewMatchingSvc() *MatchingSvc {
	baseURL := "http://localhost:9020" // Default URL, can be overridden by environment variable
	matchingSvcURL := os.Getenv("MATCHING_SVC_URL")
	if matchingSvcURL != "" {
		baseURL = matchingSvcURL
	}
	return &MatchingSvc{
		client: NewHTTPClient(),
		url:    baseURL,
	}
}

// GenerateWorkExpEmbeddings calls the matching service API to generate embeddings for a consultant's work experience
// It sends a POST request to /embeddingmodel/consultant/<consultant_id>
func (m *MatchingSvc) GenerateWorkExpEmbeddings(consultantID string) (map[string]interface{}, error) {
	if consultantID == "" {
		return nil, fmt.Errorf("consultant_id cannot be empty")
	}

	path := "/embeddingmodel/consultant/" + consultantID

	// Make POST request to the matching service
	resp, err := m.client.Post(m.url, path, nil)
	if err != nil {
		log.Printf("Error calling embedding model API: %v", err)
		return nil, err
	}
	defer resp.Close()

	// Check for HTTP error status
	if resp.GetStatusCode() != 200 {
		log.Printf("Error: received status code %d from embedding model API", resp.GetStatusCode())
		return nil, fmt.Errorf("embedding model API returned status %d", resp.GetStatusCode())
	}

	// Parse the response
	var result map[string]interface{}
	err = json.Unmarshal(resp.GetBody(), &result)
	if err != nil {
		log.Printf("Error unmarshaling response from embedding model API: %v", err)
		return nil, err
	}

	log.Printf("Successfully generated embeddings for consultant_id: %s", consultantID)
	return result, nil
}
