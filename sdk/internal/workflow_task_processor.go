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
	"fmt"
	"iter"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
)

var _ TaskProcessor = (*Conn)(nil)

func (c *Conn) ReceiveTask(ctx context.Context, includeWorkflow, includeActivity bool) (iter.Seq[*TaskToken], error) {
	if !includeWorkflow && !includeActivity {
		return nil, fmt.Errorf("at least one task type must be enabled")
	}

	consumerCtx, cancelConsumers := context.WithCancel(ctx)
	taskChannel := make(chan *TaskToken)

	type consumerHandle struct {
		consumer jetstream.Consumer
		taskType string
		handler  func(msg jetstream.Msg)
	}

	var consumers []consumerHandle

	if includeWorkflow {
		workflowTaskConsumer, err := c.EnsureConsumer(
			consumerCtx,
			c.WorkflowTaskStreamName(),
			jetstream.ConsumerConfig{
				Name:          api.WorkflowTaskWorkerConsumer,
				Durable:       api.WorkflowTaskWorkerConsumer,
				FilterSubject: c.WorkflowTaskFilterSubject(),
				AckPolicy:     jetstream.AckExplicitPolicy,
			})
		if err != nil {
			cancelConsumers()
			return nil, err
		}
		consumers = append(consumers, consumerHandle{
			consumer: workflowTaskConsumer,
			taskType: "workflow",
			handler: func(msg jetstream.Msg) {
				task := &api.WorkflowTask{}
				if err := c.converter.DeserializeBinary(msg.Data(), task); err != nil {
					msg.Term()
					return
				}
				c.enqueueTask(consumerCtx, task, msg, taskChannel)
			},
		})
	}

	if includeActivity {
		activityTaskConsumer, err := c.EnsureConsumer(
			consumerCtx,
			c.ActivityTaskStreamName(),
			jetstream.ConsumerConfig{
				Name:          api.ActivityTaskWorkerConsumer,
				Durable:       api.ActivityTaskWorkerConsumer,
				FilterSubject: c.ActivityTaskFilterSubject(),
				AckPolicy:     jetstream.AckExplicitPolicy,
			})
		if err != nil {
			cancelConsumers()
			return nil, err
		}
		consumers = append(consumers, consumerHandle{
			consumer: activityTaskConsumer,
			taskType: "activity",
			handler: func(msg jetstream.Msg) {
				task := &api.ActivityTask{}
				if err := c.converter.DeserializeBinary(msg.Data(), task); err != nil {
					msg.Term()
					return
				}
				c.enqueueTask(consumerCtx, task, msg, taskChannel)
			},
		})
	}

	var wg sync.WaitGroup

	go func() {
		wg.Wait()
		close(taskChannel)
	}()

	for _, handle := range consumers {
		wg.Add(1)
		go func(ch consumerHandle) {
			defer wg.Done()
			defer cancelConsumers()

			consumeCtx, err := ch.consumer.Consume(func(msg jetstream.Msg) {
				ch.handler(msg)
			})
			if err != nil {
				c.Logger().Error("task consumer failed", "type", ch.taskType, "error", err)
				return
			}
			defer consumeCtx.Stop()

			<-consumerCtx.Done()
		}(handle)
	}

	return func(callback func(*TaskToken) bool) {
		defer cancelConsumers()
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-taskChannel:
				if !ok {
					return
				}
				if t == nil {
					continue
				}
				switch t.Task.(type) {
				case *api.WorkflowTask, *api.ActivityTask:
					callback(t)
				default:
					t.Term(consumerCtx)
				}
			}
		}
	}, nil
}

func (c *Conn) enqueueTask(ctx context.Context, task api.Task, msg jetstream.Msg, taskChannel chan<- *TaskToken) {
	token := &TaskToken{
		Task: task,
		Ack:  msg.DoubleAck,
		Nak:  func(context.Context) error { return msg.Nak() },
		Term: func(context.Context) error { return msg.Term() },
	}

	select {
	case <-ctx.Done():
		msg.Nak()
	case taskChannel <- token:
	}
}
