package jetstreamx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Connection represents a NATS connection with JetStream capabilities.
type Connection struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func (c *Connection) Close() {
	if c.nc != nil && !c.nc.IsClosed() {
		c.nc.Close()
	}
}

// JS returns the JetStream context associated with the NATS connection.
func (c *Connection) JS() (jetstream.JetStream, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream context is not initialized")
	}
	return c.js, nil
}

// NATS returns the underlying NATS connection.
func (c *Connection) NATS() *nats.Conn {
	return c.nc
}

// IsConnected returns whether the NATS connection is currently connected.
func (c *Connection) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}

// Config is the dependency-injected interface required by the natz package.
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
func Connect(cfg Config) (*Connection, error) {
	if cfg == nil {
		return nil, fmt.Errorf("natz: nil config provided")
	}

	clientName := cfg.NATSClientName()
	if clientName == "" {
		clientName = "durablefuture"
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

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return &Connection{nc: nc, js: js}, nil
}

// Wrap upgrades an existing NATS connection with JetStream capabilities.
func Wrap(nc *nats.Conn) (*Connection, error) {
	if nc == nil {
		return nil, fmt.Errorf("natz: nil connection provided")
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return &Connection{nc: nc, js: js}, nil
}

// EnsureKV ensures that a KeyValue store with the given configuration exists.
// It creates the KV if it doesn't exist or updates it if it does.
func (c *Connection) EnsureKV(ctx context.Context, cfg jetstream.KeyValueConfig) (jetstream.KeyValue, error) {
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
func (c *Connection) EnsureStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
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
func (c *Connection) EnsureConsumer(ctx context.Context, streamName string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
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
func (c *Connection) Publish(ctx context.Context, subj string, data []byte) error {
	if err := c.nc.Publish(subj, data); err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", subj, err)
	}
	return nil
}

// PublishJS publishes a message to a JetStream subject and waits for acknowledgement.
func (c *Connection) PublishJS(ctx context.Context, subj string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	ack, err := c.js.Publish(ctx, subj, data, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to publish JetStream message to subject %s: %w", subj, err)
	}
	return ack, nil
}

// Subscribe creates a subscription to a subject using basic NATS.
func (c *Connection) SubscribeAsync(subj string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.nc.Subscribe(subj, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subj, err)
	}
	return sub, nil
}

// Subscribe creates a subscription to a subject using basic NATS.
func (c *Connection) SubscribeSync(subj string) (*nats.Subscription, error) {
	sub, err := c.nc.SubscribeSync(subj)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subj, err)
	}
	return sub, nil
}

// QueueSubscribe creates a queue subscription to a subject using basic NATS.
func (c *Connection) QueueSubscribe(subj, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := c.nc.QueueSubscribe(subj, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to subject %s with queue %s: %w", subj, queue, err)
	}
	return sub, nil
}

// FetchMessages fetches a batch of messages from a consumer.
func (c *Connection) FetchMessages(ctx context.Context, consumer jetstream.Consumer, batchSize int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
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
func (c *Connection) WatchKV(ctx context.Context, bucket, key string, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
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
func (c *Connection) Set(ctx context.Context, bucket, key string, value []byte) (uint64, error) {
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
func (c *Connection) Get(ctx context.Context, bucket, key string) (jetstream.KeyValueEntry, error) {
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
