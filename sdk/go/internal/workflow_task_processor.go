package internal

import (
	"context"
	"iter"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
)

var _ TaskProcessor = (*Conn)(nil)

func (c *Conn) ReceiveTask(ctx context.Context) (iter.Seq[*TaskToken], error) {
	consumerCtx, cancelConsumers := context.WithCancel(ctx)
	defer cancelConsumers()

	workflowTaskConsumer, err := c.EnsureConsumer(
		consumerCtx,
		c.TaskStreamName(),
		jetstream.ConsumerConfig{
			FilterSubject: c.WorkflowTaskFilterSubject(),
		})
	if err != nil {
		cancelConsumers()
		return nil, err
	}

	activityTaskConsumer, err := c.EnsureConsumer(
		consumerCtx,
		c.TaskStreamName(),
		jetstream.ConsumerConfig{
			FilterSubject: c.ActivityTaskFilterSubject(),
		})
	if err != nil {
		cancelConsumers()
		return nil, err
	}

	taskChannel := make(chan *TaskToken)
	var wg sync.WaitGroup

	go func() {
		wg.Wait()
		close(taskChannel)
	}()

	wg.Go(func() {
		defer cancelConsumers()

		consumeCtx, err := workflowTaskConsumer.Consume(func(msg jetstream.Msg) {
			var task api.WorkflowTask
			err := c.converter.DeserializeBinary(msg.Data(), &task)
			if err != nil {
				// kill poison pill
				msg.Term()
			}

			taskChannel <- &TaskToken{
				Task: &task,
				Ack:  msg.DoubleAck,
				Nak:  func(ctx context.Context) error { return msg.Nak() },
				Term: func(ctx context.Context) error { return msg.Term() },
			}
		})
		if err != nil {
			return
		}
		defer consumeCtx.Stop()

		<-consumerCtx.Done()
	})

	wg.Go(func() {
		defer cancelConsumers()

		consumeCtx, err := activityTaskConsumer.Consume(func(msg jetstream.Msg) {
			var task api.ActivityTask
			err := c.converter.DeserializeBinary(msg.Data(), &task)
			if err != nil {
				// kill poison pill
				msg.Term()
			}

			taskChannel <- &TaskToken{
				Task: &task,
				Ack:  msg.DoubleAck,
				Nak:  func(ctx context.Context) error { return msg.Nak() },
				Term: func(ctx context.Context) error { return msg.Term() },
			}
		})
		if err != nil {
			return
		}
		defer consumeCtx.Stop()

		<-consumerCtx.Done()
	})

	return func(callback func(*TaskToken) bool) {
		defer cancelConsumers()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-taskChannel:
				switch t.Task.(type) {
				case *api.WorkflowTask, *api.ActivityTask:
					callback(t)
				default:
					// poison pill
					t.Term(consumerCtx)
				}
			}
		}
	}, nil
}
