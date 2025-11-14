// Copyright 2025 Nguyen Nhat Nguyen
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

package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
)

type (
	IdentifierManager interface {
		Namespace() string
		WorkflowTaskStreamName() string
		ActivityTaskStreamName() string

		WorkflowTaskFilterSubject() string
		ActivityTaskFilterSubject() string
	}

	idManager struct {
		ns string
	}
)

func (i *idManager) Namespace() string {
	return i.ns
}

func (i *idManager) WorkflowTaskStreamName() string {
	if i.ns == "" {
		return api.WorkflowTasksStream
	}
	return i.ns + "_" + api.WorkflowTasksStream
}

func (i *idManager) ActivityTaskStreamName() string {
	if i.ns == "" {
		return api.ActivityTasksStream
	}
	return i.ns + "_" + api.ActivityTasksStream
}

func (i *idManager) WorkflowTaskFilterSubject() string {
	if i.ns == "" {
		return api.WorkflowTasksFilterSubjectPattern
	}
	return fmt.Sprintf("%s.%s", i.ns, "tasks.workflow")
}

func (i *idManager) ActivityTaskFilterSubject() string {
	if i.ns == "" {
		return api.ActivityTasksFilterSubjectPattern
	}
	return fmt.Sprintf("%s.%s.*.%s", i.ns, "tasks", "activity")
}

// Conn represents a NATS connection with JetStream capabilities tailored for the SDK.
type Conn struct {
	nc        *nats.Conn
	js        jetstream.JetStream
	converter serde.BinarySerde

	IdentifierManager
	logger *slog.Logger
}

func from(nc *nats.Conn, namespace string, conv serde.BinarySerde) (*Conn, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	namespace = strings.TrimSpace(namespace)
	if conv == nil {
		conv = &serde.MsgpackSerde{} // Default to MessagePack for better performance
	}
	return &Conn{
		nc:                nc,
		js:                js,
		converter:         conv,
		IdentifierManager: &idManager{ns: namespace},
	}, nil
}

func (c *Conn) Close() {
	if c.nc != nil && !c.nc.IsClosed() {
		c.nc.Close()
	}
}

func (c *Conn) SetLogger(l *slog.Logger) {
	c.logger = defaultLogger(l)
}

func (c *Conn) Logger() *slog.Logger {
	if c == nil {
		return slog.Default()
	}
	return defaultLogger(c.logger)
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

// Config is the dependency-injected interface required for establishing connections.
type Config interface {
	Endpoint() string
	NATSMaxReconnects() int
	NATSReconnectWait() time.Duration
	NATSDrainTimeout() time.Duration
	NATSPingInterval() time.Duration
	NATSMaxPingsOut() int
	// Optional human readable client name; may return empty.
	NATSClientName() string
}

// Connect establishes a connection to NATS with the given configuration.
func Connect(cfg Config, namespace string, conv serde.BinarySerde) (*Conn, error) {
	if cfg == nil {
		return nil, fmt.Errorf("natz: nil config provided")
	}

	clientName := cfg.NATSClientName()
	if clientName == "" {
		clientName = "durablefuture-sdk"
	}
	opts := []nats.Option{
		nats.Name(clientName),
		nats.MaxReconnects(cfg.NATSMaxReconnects()),
		nats.ReconnectWait(cfg.NATSReconnectWait()),
		nats.DrainTimeout(cfg.NATSDrainTimeout()),
		nats.PingInterval(cfg.NATSPingInterval()),
		nats.MaxPingsOutstanding(cfg.NATSMaxPingsOut()),
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

	nc, err := nats.Connect(cfg.Endpoint(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", cfg.Endpoint(), err)
	}
	conn, err := from(nc, namespace, conv)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func wrapExisting(nc *nats.Conn, namespace string, conv serde.BinarySerde) (*Conn, error) {
	if nc == nil {
		return nil, fmt.Errorf("natz: nil connection provided")
	}
	return from(nc, namespace, conv)
}

// EnsureKV ensures that a KeyValue store with the given configuration exists.
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
		return nil, fmt.Errorf("failed to get stream %s info: %w", cfg.Name, err)
	}

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

// Publish publishes a message to a subject using basic NATS.
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
func (c *Conn) Get(ctx context.Context, bucket, key string) (jetstream.KeyValueEntry, error) {
	kv, err := c.js.KeyValue(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket '%s': %w", bucket, err)
	}

	entry, err := kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return entry, nil
}
