package clientSide

// HTTPMethod defines the HTTP method type
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
)

// Request defines the interface for HTTP requests
type Request interface {
	// SetURL sets the base URL for the request
	SetURL(url string) Request

	// SetPath sets the endpoint path
	SetPath(path string) Request

	// SetMethod sets the HTTP method
	SetMethod(method HTTPMethod) Request

	// SetBody sets the request body
	SetBody(body []byte) Request

	// SetHeaders sets custom headers
	SetHeaders(headers map[string]string) Request

	// Execute sends the HTTP request and returns the response
	Execute() (Response, error)
}

// Response defines the interface for HTTP responses
type Response interface {
	// GetStatusCode returns the HTTP status code
	GetStatusCode() int

	// GetBody returns the response body as bytes
	GetBody() []byte

	// GetHeaders returns all response headers
	GetHeaders() map[string][]string

	// GetHeader returns a specific header value
	GetHeader(key string) string

	// Close closes the response body
	Close() error
}

// HTTPClient defines the interface for an HTTP client
type HTTPClient interface {
	// NewRequest creates a new HTTP request
	NewRequest() Request

	// Get performs a GET request to the specified URL and path
	Get(url, path string) (Response, error)

	// Post performs a POST request to the specified URL and path with body
	Post(url, path string, body []byte) (Response, error)

	// Put performs a PUT request to the specified URL and path with body
	Put(url, path string, body []byte) (Response, error)

	// Delete performs a DELETE request to the specified URL and path
	Delete(url, path string) (Response, error)
}
