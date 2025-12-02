package command

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
	jetstreamx "github.com/ngnhng/durablefuture/internal/server/infra/jetstream"
)

type Handler struct {
	conv serde.BinarySerde
	conn *jetstreamx.Connection
}

func NewHandler(conn *jetstreamx.Connection, conv serde.BinarySerde) *Handler {
	return &Handler{
		conv: conv,
		conn: conn,
	}
}

// sendErrorReply sends an error reply using the configured serializer
func (h *Handler) sendErrorReply(msg *nats.Msg, workflowID, errorMsg string) {
	reply, err := h.conv.SerializeBinary(api.StartWorkflowReply{
		WorkflowID: workflowID,
		Error:      errorMsg,
	})
	if err != nil {
		// If serialization fails, send a minimal JSON response as last resort
		fallback := fmt.Sprintf(`{"error":"serialization failed: %s","workflow_id":""}`, err.Error())
		msg.Respond([]byte(fallback))
		return
	}
	msg.Respond(reply)
}

func (h *Handler) HandleRequest(msg *nats.Msg) {
	slog.Debug("handler entry", "subject", msg.Subject, "reply", msg.Reply)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in request handler", "error", r)
			if err := msg.Term(); err != nil {
				slog.Error("failed to nak message after panic", "error", err)
			}
		}
	}()

	var cmd api.Command
	if err := h.conv.DeserializeBinary(msg.Data, &cmd); err != nil {
		slog.Error("failed to deserialize command", "error", err)
		msg.Term()
		return
	}

	switch cmd.CommandType {
	case api.StartWorkflowCommand:
		{
			var data api.StartWorkflowAttributes
			if err := h.conv.DeserializeBinary(cmd.Attributes, &data); err != nil {
				slog.Debug("failed to unmarshal start workflow attributes", "error", err)
				h.sendErrorReply(msg, "", "failed to parse request attributes: "+err.Error())
				return
			}
			slog.Debug("received request data", "workflow_fn", data.WorkflowFnName, "input_count", len(data.Input))
			workflowID, _ := uuid.NewV7()
			idStr := workflowID.String()
			slog.Debug("generated workflow ID", "workflow_id", idStr)

			event := api.WorkflowStarted{
				ID:             api.WorkflowID(workflowID.String()),
				WorkflowFnName: data.WorkflowFnName,
				Input:          data.Input,
			}

			js, err := h.conn.JS()
			if err != nil {
				slog.Debug("failed to get JetStream context", "error", err)
				h.sendErrorReply(msg, "", "internal server error: failed to get JetStream context: "+err.Error())
				return
			}

			reply, err := h.conv.SerializeBinary(api.StartWorkflowReply{WorkflowID: workflowID.String()})
			if err != nil {
				slog.Debug("failed to convert reply to bytes", "error", err)
				// Send a basic JSON error response
				errorReply := fmt.Sprintf(`{"error":"failed to serialize reply: %s","workflow_id":""}`, err.Error())
				slog.Debug("sending error reply", "reply", errorReply)
				msg.Respond([]byte(errorReply))
				return
			}

			// Store input arguments in KV store using workflow function name as key
			kv, err := js.KeyValue(context.Background(), api.WorkflowInputBucket)
			if err != nil {
				slog.Debug("failed to get KV store", "error", err)
				h.sendErrorReply(msg, "", "internal server error: failed to get KV store: "+err.Error())
				return
			}

			inputArgsData, err := h.conv.SerializeBinary(data.Input)
			if err != nil {
				slog.Debug("failed to serialize input args", "error", err)
				h.sendErrorReply(msg, "", "internal server error: failed to serialize input args: "+err.Error())
				return
			}

			key := api.WorkflowInputKey(data.WorkflowFnName, api.WorkflowID(workflowID.String()))
			_, err = kv.Put(context.Background(), key, inputArgsData)
			if err != nil {
				slog.Debug("failed to store input args in KV", "error", err)
				h.sendErrorReply(msg, "", "internal server error: failed to store input args: "+err.Error())
				return
			}
			slog.Debug("stored input args in KV", "workflow_fn", data.WorkflowFnName, "key", key)

			subject := fmt.Sprintf(api.HistoryPublishSubjectPattern, idStr)
			eventBytes, _ := h.conv.SerializeBinary(event)
			startEventMsg := &nats.Msg{
				Subject: subject,
				Data:    eventBytes,
				Header: nats.Header{
					api.ChronicleEventNameHeader:        []string{event.EventName()},
					api.ChronicleAggregateVersionHeader: []string{strconv.FormatUint(1, 10)},
				},
			}
			_, err = js.PublishMsg(context.Background(), startEventMsg)
			if err != nil {
				slog.Debug("failed to publish event to history", "error", err)
				h.sendErrorReply(msg, "", "internal server error: "+err.Error())
				return
			}
			slog.Debug("published event to history", "workflow_id", idStr, "event", event.EventName())

			slog.Debug("sending response", "reply_subject", msg.Reply, "reply_size", len(reply))
			err = msg.Respond(reply)
			if err != nil {
				slog.Debug("failed to send response", "error", err)
				return
			}

			slog.Debug("published reply successfully")

			return

		}
	default:
		msg.Term()
	}
}

func RunProcessor(ctx context.Context, conn *jetstreamx.Connection, handler *Handler) error {
	sub, err := conn.QueueSubscribe(
		api.CommandRequestSubjectPattern,
		api.ManagerCommandProcessorsConsumer,
		handler.HandleRequest,
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}
