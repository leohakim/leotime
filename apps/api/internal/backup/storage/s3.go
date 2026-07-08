package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKeyID  string
	SecretKey    string
	UsePathStyle bool
}

type S3Client struct {
	client *s3.Client
	bucket string
	compat *compatS3Client
}

type compatS3Client struct {
	bucket       string
	endpoint     *url.URL
	region       string
	credentials  credentials.StaticCredentialsProvider
	httpClient   *http.Client
	usePathStyle bool
}

func NewS3Client(ctx context.Context, cfg S3Config) (*S3Client, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("access credentials are required")
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}

	accessKeyID := strings.TrimSpace(cfg.AccessKeyID)
	secretKey := strings.TrimSpace(cfg.SecretKey)

	endpoint := NormalizeEndpoint(cfg.Endpoint)
	usePathStyle := EffectivePathStyle(endpoint, cfg.UsePathStyle)
	compatMode := IsCustomEndpoint(endpoint)

	loadCfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretKey,
			"",
		),
	}

	if compatMode {
		compatEndpoint, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse endpoint: %w", err)
		}
		return &S3Client{
			bucket: cfg.Bucket,
			compat: &compatS3Client{
				bucket:       cfg.Bucket,
				endpoint:     compatEndpoint,
				region:       region,
				credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
				httpClient:   http.DefaultClient,
				usePathStyle: usePathStyle,
			},
		}, nil
	}

	options := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = usePathStyle
		},
	}
	if endpoint != "" {
		options = append(options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	client := s3.NewFromConfig(loadCfg, options...)
	_ = ctx

	return &S3Client{client: client, bucket: cfg.Bucket}, nil
}

func NormalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	if endpoint == "" {
		return ""
	}
	endpoint = strings.TrimRight(endpoint, "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return endpoint
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path != "" && parsed.Path != "/" {
		return parsed.String()
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/")
}

func IsCustomEndpoint(endpoint string) bool {
	endpoint = strings.ToLower(strings.TrimSpace(endpoint))
	if endpoint == "" {
		return false
	}
	return !strings.Contains(endpoint, "amazonaws.com")
}

func EffectivePathStyle(endpoint string, configured bool) bool {
	if configured {
		return true
	}
	return IsCustomEndpoint(endpoint)
}

func EndpointUsesHTTP(endpoint string) bool {
	endpoint = NormalizeEndpoint(endpoint)
	if endpoint == "" {
		return false
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return strings.HasPrefix(strings.ToLower(endpoint), "http://")
	}
	return parsed.Scheme == "http"
}

func (c *S3Client) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	if c.compat != nil {
		return c.compat.Put(ctx, key, body, contentType)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (c *S3Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if c.compat != nil {
		return c.compat.Get(ctx, key)
	}
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	return output.Body, nil
}

func (c *S3Client) Delete(ctx context.Context, key string) error {
	if c.compat != nil {
		return c.compat.Delete(ctx, key)
	}
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

func (c *S3Client) List(ctx context.Context, prefix string) ([]Object, error) {
	if c.compat != nil {
		return c.compat.List(ctx, prefix)
	}
	output, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}

	objects := make([]Object, 0, len(output.Contents))
	for _, item := range output.Contents {
		if item.Key == nil {
			continue
		}
		objects = append(objects, Object{
			Key:          aws.ToString(item.Key),
			SizeBytes:    aws.ToInt64(item.Size),
			LastModified: objectTime(item.LastModified),
		})
	}

	return objects, nil
}

func (c *compatS3Client) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	resp, err := c.do(ctx, http.MethodPut, key, nil, body, contentType)
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *compatS3Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := c.do(ctx, http.MethodGet, key, nil, nil, "")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	return resp.Body, nil
}

func (c *compatS3Client) Delete(ctx context.Context, key string) error {
	resp, err := c.do(ctx, http.MethodDelete, key, nil, nil, "")
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *compatS3Client) List(ctx context.Context, prefix string) ([]Object, error) {
	query := url.Values{}
	query.Set("list-type", "2")
	if strings.TrimSpace(prefix) != "" {
		query.Set("prefix", prefix)
	}
	resp, err := c.do(ctx, http.MethodGet, "", query, nil, "")
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Contents []struct {
			Key          string `xml:"Key"`
			SizeBytes    int64  `xml:"Size"`
			LastModified string `xml:"LastModified"`
		} `xml:"Contents"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}

	objects := make([]Object, 0, len(payload.Contents))
	for _, item := range payload.Contents {
		if strings.TrimSpace(item.Key) == "" {
			continue
		}
		objects = append(objects, Object{
			Key:          item.Key,
			SizeBytes:    item.SizeBytes,
			LastModified: parseS3Time(item.LastModified),
		})
	}
	return objects, nil
}

func (c *compatS3Client) do(ctx context.Context, method, key string, query url.Values, body io.Reader, contentType string) (*http.Response, error) {
	target := *c.endpoint
	target.Path = c.objectPath(key)
	target.RawQuery = ""
	if query != nil {
		target.RawQuery = query.Encode()
	}
	if !c.usePathStyle {
		target.Host = c.bucket + "." + target.Host
	}

	payloadBody, contentLength, payloadHash, err := preparePayload(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, target.String(), payloadBody)
	if err != nil {
		return nil, err
	}
	req.ContentLength = contentLength
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	creds, err := c.credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieve credentials: %w", err)
	}
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "s3", c.region, time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, parseS3Error(resp)
	}
	return resp, nil
}

func (c *compatS3Client) objectPath(key string) string {
	base := strings.TrimSuffix(c.endpoint.Path, "/")
	if c.usePathStyle {
		if key == "" {
			return ensureLeadingSlash(base + "/" + c.bucket)
		}
		return ensureLeadingSlash(base + "/" + c.bucket + "/" + escapeS3Path(key))
	}
	if key == "" {
		return ensureLeadingSlash(base)
	}
	return ensureLeadingSlash(base + "/" + escapeS3Path(key))
}

func ensureLeadingSlash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func escapeS3Path(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func preparePayload(body io.Reader) (io.Reader, int64, string, error) {
	if body == nil {
		hash := sha256.Sum256(nil)
		return http.NoBody, 0, hex.EncodeToString(hash[:]), nil
	}

	if seeker, ok := body.(io.ReadSeeker); ok {
		hash := sha256.New()
		size, err := io.Copy(hash, seeker)
		if err != nil {
			return nil, 0, "", fmt.Errorf("hash payload: %w", err)
		}
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, 0, "", fmt.Errorf("rewind payload: %w", err)
		}
		return seeker, size, hex.EncodeToString(hash.Sum(nil)), nil
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, 0, "", fmt.Errorf("read payload: %w", err)
	}
	hash := sha256.Sum256(data)
	return bytes.NewReader(data), int64(len(data)), hex.EncodeToString(hash[:]), nil
}

func parseS3Error(resp *http.Response) error {
	var payload struct {
		Code      string `xml:"Code"`
		Message   string `xml:"Message"`
		RequestID string `xml:"RequestId"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := xml.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Code) != "" {
		return fmt.Errorf("https response error StatusCode: %d, RequestID: %s, api error %s: %s", resp.StatusCode, payload.RequestID, payload.Code, payload.Message)
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	return fmt.Errorf("https response error StatusCode: %d, message: %s", resp.StatusCode, message)
}

func parseS3Time(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func objectTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

var _ Client = (*S3Client)(nil)
