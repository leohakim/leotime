package storage

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"time"
)

type Object struct {
	Key          string
	SizeBytes    int64
	LastModified time.Time
}

type Client interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]Object, error)
}

type MemoryClient struct {
	Objects    map[string][]byte
	ModifiedAt map[string]time.Time
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{
		Objects:    map[string][]byte{},
		ModifiedAt: map[string]time.Time{},
	}
}

func (c *MemoryClient) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	_ = ctx
	_ = contentType
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	c.Objects[key] = data
	if _, ok := c.ModifiedAt[key]; !ok {
		c.ModifiedAt[key] = time.Now().UTC()
	}
	return nil
}

func (c *MemoryClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	_ = ctx
	data, ok := c.Objects[key]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (c *MemoryClient) Delete(ctx context.Context, key string) error {
	_ = ctx
	delete(c.Objects, key)
	return nil
}

func (c *MemoryClient) List(ctx context.Context, prefix string) ([]Object, error) {
	_ = ctx
	objects := make([]Object, 0)
	for key, data := range c.Objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		modified := c.ModifiedAt[key]
		if modified.IsZero() {
			modified = time.Now().UTC()
		}
		objects = append(objects, Object{
			Key:          key,
			SizeBytes:    int64(len(data)),
			LastModified: modified,
		})
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})
	return objects, nil
}
