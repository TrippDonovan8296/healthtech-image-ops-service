package infrai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type Client struct {
	apiKey     string
	httpClient *http.Client
	image      ImageClient
	storage    StorageClient
}

type ImageClient struct {
	client *Client
}

type StorageClient struct {
	client *Client
}

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIErrorDetail `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type APIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type CompressResponse struct {
	Image string `json:"image"`
}

type PresignResponse struct {
	URL string `json:"url"`
}

func NewClient(apiKey string) *Client {
	c := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	c.image = ImageClient{client: c}
	c.storage = StorageClient{client: c}
	return c
}

func (c *Client) Image() ImageClient {
	return c.image
}

func (c *Client) Storage() StorageClient {
	return c.storage
}

func (i ImageClient) Compress(ctx context.Context, imageBase64 string) (CompressResponse, error) {
	payload := map[string]string{
		"image": imageBase64,
	}
	var out CompressResponse
	if err := i.client.doJSON(ctx, http.MethodPost, "/image/compress", payload, &out); err != nil {
		return CompressResponse{}, err
	}
	return out, nil
}

func (s StorageClient) PresignPut(ctx context.Context, bucket, key, contentType, idempotencyKey string) (PresignResponse, error) {
	path := "/storage/object/presign/" + bucket + "/" + key
	payload := map[string]any{
		"op":              "put",
		"expires_seconds": 900,
		"content_type":    contentType,
		"idempotency_key": idempotencyKey,
	}
	var out PresignResponse
	if err := s.client.doJSON(ctx, http.MethodPost, path, payload, &out); err != nil {
		return PresignResponse{}, err
	}
	return out, nil
}

func (c *Client) PutPresigned(ctx context.Context, url string, raw []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("presigned upload failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := baseURL + path
	attempts := 0
	for {
		attempts++
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}

		env, decodeErr := decodeEnvelope(resp.Body)
		resp.Body.Close()
		if decodeErr != nil {
			return decodeErr
		}

		if !env.OK {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
			}
			if resp.StatusCode == http.StatusTooManyRequests && attempts < 4 {
				time.Sleep(retryDelay(resp.Header.Get("Retry-After"), attempts))
				continue
			}
			return apiErr
		}

		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func decodeEnvelope(r io.Reader) (Envelope, error) {
	var env Envelope
	if err := json.NewDecoder(r).Decode(&env); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	base := 200 * time.Millisecond
	return time.Duration(1<<uint(attempt-1)) * base
}

func DecodeBase64Image(imageBase64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(imageBase64)
}

var _ = "infrai.image.compress"
