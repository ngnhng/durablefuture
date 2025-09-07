// Copyright 2025 Nguyen-Nhat Nguyen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package natz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Conn represents a NATS connection with JetStream capabilities.
type Conn struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func (c *Conn) Close() {
	if c.nc != nil && !c.nc.IsClosed() {
		c.nc.Close()
	}
}

// JS returns the JetStream context associated with the NATS connection.
func (c *Conn) JS() (jetstream.JetStream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream context is not initialized")
	}
	return c.js, nil
}

// NATS returns the underlying NATS connection.
func (c *Conn) NATS() *nats.Conn {
	return c.nc
}

// IsConnected returns whether the NATS connection is currently connected.
func (c *Conn) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}

// Config holds NATS connection configuration.
type Config struct {
	URL           string
	MaxReconnects int
	ReconnectWait time.Duration
	Name          string
	DrainTimeout  time.Duration
	PingInterval  time.Duration
	MaxPingsOut   int
}

// DefaultConfig returns a default configuration for NATS connection.
func DefaultConfig() Config {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL // Default to localhost if not set
	}

	return Config{
		URL:           natsURL,
		MaxReconnects: -1, // Reconnect forever
		ReconnectWait: 2 * time.Second,
		DrainTimeout:  30 * time.Second,
		PingInterval:  2 * time.Minute,
		MaxPingsOut:   2,
	}
}

// Connect establishes a connection to NATS with the given configuration.
func Connect(cfg Config) (*Conn, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DrainTimeout(cfg.DrainTimeout),
		nats.PingInterval(cfg.PingInterval),
		nats.MaxPingsOutstanding(cfg.MaxPingsOut),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			fmt.Printf("NATS disconnected: %v\n", err)
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			fmt.Println("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", cfg.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return &Conn{nc: nc, js: js}, nil
}

// EnsureKV ensures that a KeyValue store with the given configuration exists.
// It creates the KV if it doesn't exist or updates it if it does.
func (c *Conn) EnsureKV(ctx context.Context, cfg jetstream.KeyValueConfig) (jetstream.KeyValue, error) {
	kv, err := c.js.KeyValue(ctx, cfg.Bucket)
	if err != nil || kv == nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			kv, err := c.js.CreateKeyValue(ctx, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create new KV: %v, %w", cfg.Bucket, err)
			}
			return kv, nil
		}
		return nil, fmt.Errorf("failed to ensure KV: %v, %w", cfg.Bucket, err)
	}

	updatedKV, err := c.js.UpdateKeyValue(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update KV: %v, %w", cfg.Bucket, err)
	}

	return updatedKV, err
}

// EnsureStream ensures that a stream with the given configuration exists.
// It creates the stream if it doesn't exist or updates it if it does.
func (c *Conn) EnsureStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	stream, err := c.js.Stream(ctx, cfg.Name)
	if err != nil || stream == nil {
		if errors.Is(err, jetstream.ErrStreamNotFound) {
			stream, err = c.js.CreateStream(ctx, cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create stream %s: %w", cfg.Name, err)
			}
			return stream, nil
		}
		// Other error fetching stream info
		return nil, fmt.Errorf("failed to get stream %s info: %w", cfg.Name, err)
	}

	// Stream exists, update it (idempotent)
	// Note: UpdateStream might return specific errors if the update is incompatible.
	streamInfo, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info %s: %w", cfg.Name, err)
	}
	if streamInfo.Config.Retention == jetstream.WorkQueuePolicy {
		cfg.Retention = jetstream.WorkQueuePolicy
	} else {
		cfg.Retention = streamInfo.Config.Retention
	}

	updatedStream, err := c.js.UpdateStream(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update stream %s: %w", cfg.Name, err)
	}
	return updatedStream, nil
}

// EnsureConsumer ensures that a consumer with the given configuration exists on the specified stream.
// It creates the consumer if it doesn't exist or updates it if it does.
func (c *Conn) EnsureConsumer(ctx context.Context, streamName string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	stream, err := c.js.Stream(ctx, streamName)
	if err != nil || stream == nil {
		return nil, fmt.Errorf("failed to get stream %s for consumer creation: %w", streamName, err)
	}

	consumer, err := stream.Consumer(ctx, cfg.Name)
	if err != nil || consumer == nil {
		consumer, err = stream.CreateOrUpdateConsumer(ctx, cfg)
		if err != nil || consumer == nil {
			return nil, fmt.Errorf("failed to create/update consumer %s on stream %s: %w", cfg.Name, streamName, err)
		}
	}
	return consumer, nil
}

// Publish publishes a message to a subject using basic NATS (not JetStream).
// This is useful for simple fire-and-forget messaging.
func (c *Conn) Publish(ctx context.Context, subj string, data []byte) error {
	if err := c.nc.Publish(subj, data); err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", subj, err)
	}
	return nil
}

// PublishJS publishes a message to a JetStream subject and waits for acknowledgement.
func (c *Conn) PublishJS(ctx context.Context, subj string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	ack, err := c.js.Publish(ctx, subj, data, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to publish JetStream message to subject %s: %w", subj, err)
	}
	return ack, nil
}

// Subscribe creates a subscription to a subject using basic NATS.
func (c *Conn) SubscribeAsync(subj string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.nc.Subscribe(subj, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subj, err)
	}
	return sub, nil
}

// Subscribe creates a subscription to a subject using basic NATS.
func (c *Conn) SubscribeSync(subj string) (*nats.Subscription, error) {
	sub, err := c.nc.SubscribeSync(subj)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subj, err)
	}
	return sub, nil
}

// QueueSubscribe creates a queue subscription to a subject using basic NATS.
func (c *Conn) QueueSubscribe(subj, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.nc.QueueSubscribe(subj, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to subject %s with queue %s: %w", subj, queue, err)
	}
	return sub, nil
}

// FetchMessages fetches a batch of messages from a consumer.
func (c *Conn) FetchMessages(ctx context.Context, consumer jetstream.Consumer, batchSize int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	msgs, err := consumer.Fetch(batchSize, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %d messages: %w", batchSize, err)
	}
	return msgs, nil
}

// WatchKV creates a watcher for a given key or key pattern within a bucket.
// The key can include wildcards (*, >).
// The returned KeyWatcher must be stopped by the caller when no longer needed
// to clean up resources, e.g., using `defer watcher.Stop()`.
func (c *Conn) WatchKV(ctx context.Context, bucket, key string, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
	kv, err := c.js.KeyValue(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket '%s': %w", bucket, err)
	}

	watcher, err := kv.Watch(ctx, key, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher for key '%s' in bucket '%s': %w", key, bucket, err)
	}

	return watcher, nil
}

// Set stores a key-value pair in the specified bucket.
// It returns the revision number of the key.
func (c *Conn) Set(ctx context.Context, bucket, key string, value []byte) (uint64, error) {
	kv, err := c.js.KeyValue(ctx, bucket)
	if err != nil {
		return 0, fmt.Errorf("failed to get KV bucket '%s': %w", bucket, err)
	}

	rev, err := kv.Put(ctx, key, value)
	if err != nil {
		return 0, fmt.Errorf("failed to put key '%s' in bucket '%s': %w", key, bucket, err)
	}
	return rev, nil
}

// Get retrieves a key-value entry from the specified bucket.
// Returns jetstream.ErrKeyNotFound if the key does not exist.
func (c *Conn) Get(ctx context.Context, bucket, key string) (jetstream.KeyValueEntry, error) {
	kv, err := c.js.KeyValue(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket '%s': %w", bucket, err)
	}

	entry, err := kv.Get(ctx, key)
	if err != nil {
		// The error will be jetstream.ErrKeyNotFound if the key doesn't exist.
		return nil, err
	}
	return entry, nil
}
