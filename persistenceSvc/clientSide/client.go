package clientSide

import (
	"bytes"
	"io"
	"net/http"
)

// RequestImpl implements the Request interface
type RequestImpl struct {
	url     string
	path    string
	method  HTTPMethod
	body    []byte
	headers map[string]string
	client  *http.Client
}

// ResponseImpl implements the Response interface
type ResponseImpl struct {
	statusCode int
	body       []byte
	headers    map[string][]string
	closer     io.Closer
}

// HTTPClientImpl implements the HTTPClient interface
type HTTPClientImpl struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client instance
func NewHTTPClient() HTTPClient {
	return &HTTPClientImpl{
		client: &http.Client{},
	}
}

// NewRequest creates a new HTTP request
func (c *HTTPClientImpl) NewRequest() Request {
	return &RequestImpl{
		method:  GET,
		headers: make(map[string]string),
		client:  c.client,
	}
}

// SetURL sets the base URL for the request
func (r *RequestImpl) SetURL(url string) Request {
	r.url = url
	return r
}

// SetPath sets the endpoint path
func (r *RequestImpl) SetPath(path string) Request {
	r.path = path
	return r
}

// SetMethod sets the HTTP method
func (r *RequestImpl) SetMethod(method HTTPMethod) Request {
	r.method = method
	return r
}

// SetBody sets the request body
func (r *RequestImpl) SetBody(body []byte) Request {
	r.body = body
	return r
}

// SetHeaders sets custom headers
func (r *RequestImpl) SetHeaders(headers map[string]string) Request {
	r.headers = headers
	return r
}

// Execute sends the HTTP request and returns the response
func (r *RequestImpl) Execute() (Response, error) {
	fullURL := r.url + r.path

	// Create HTTP request
	req, err := http.NewRequest(string(r.method), fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Add body if present
	if len(r.body) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(r.body))
		req.ContentLength = int64(len(r.body))
	}

	// Add headers
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}

	// Default content type for JSON
	if req.Header.Get("Content-Type") == "" && len(r.body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	return &ResponseImpl{
		statusCode: resp.StatusCode,
		body:       bodyBytes,
		headers:    resp.Header,
		closer:     resp.Body,
	}, nil
}

// GetStatusCode returns the HTTP status code
func (r *ResponseImpl) GetStatusCode() int {
	return r.statusCode
}

// GetBody returns the response body as bytes
func (r *ResponseImpl) GetBody() []byte {
	return r.body
}

// GetHeaders returns all response headers
func (r *ResponseImpl) GetHeaders() map[string][]string {
	return r.headers
}

// GetHeader returns a specific header value
func (r *ResponseImpl) GetHeader(key string) string {
	headers := r.headers[key]
	if len(headers) > 0 {
		return headers[0]
	}
	return ""
}

// Close closes the response body
func (r *ResponseImpl) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// Get performs a GET request to the specified URL and path
func (c *HTTPClientImpl) Get(url, path string) (Response, error) {
	return c.NewRequest().
		SetURL(url).
		SetPath(path).
		SetMethod(GET).
		Execute()
}

// Post performs a POST request to the specified URL and path with body
func (c *HTTPClientImpl) Post(url, path string, body []byte) (Response, error) {
	return c.NewRequest().
		SetURL(url).
		SetPath(path).
		SetMethod(POST).
		SetBody(body).
		Execute()
}

// Put performs a PUT request to the specified URL and path with body
func (c *HTTPClientImpl) Put(url, path string, body []byte) (Response, error) {
	return c.NewRequest().
		SetURL(url).
		SetPath(path).
		SetMethod(PUT).
		SetBody(body).
		Execute()
}

// Delete performs a DELETE request to the specified URL and path
func (c *HTTPClientImpl) Delete(url, path string) (Response, error) {
	return c.NewRequest().
		SetURL(url).
		SetPath(path).
		SetMethod(DELETE).
		Execute()
}
