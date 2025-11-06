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

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gofrs/uuid/v5"
	"github.com/nats-io/nats.go"
	"github.com/ngnhng/durablefuture/api"
	constant "github.com/ngnhng/durablefuture/api"
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

func (h *Handler) HandleRequest(msg *nats.Msg) {
	slog.Debug(fmt.Sprintf("HANDLER ENTRY: Subject=%s, Reply=%s", msg.Subject, msg.Reply))

	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC in request handler", "error", r)
			if err := msg.Term(); err != nil {
				slog.Error("Failed to nak message after panic", "error", err)
			}
		}
	}()

	var cmd api.Command
	err := json.Unmarshal(msg.Data, &cmd)
	if err != nil {
		slog.Error("unmarshal error", "error", err)
		msg.Term()
		return
	}

	switch cmd.CommandType {
	case api.StartWorkflowCommand:
		{
			var data api.StartWorkflowAttributes
			if err := h.conv.DeserializeBinary(cmd.Attributes, &data); err != nil {
				slog.Debug(fmt.Sprintf("failed to unmarshal start workflow attributes: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					WorkflowID: "",
					Error:      "failed to parse request attributes: " + err.Error()})

				slog.Debug(fmt.Sprintf("publishing error reply for unmarshal failure: %v", string(reply)))
				msg.Respond(reply)
				return
			}
			slog.Debug(fmt.Sprintf("request data: %v", data))
			workflowID, _ := uuid.NewV7()
			slog.Debug(fmt.Sprintf("generated workflow ID: %v", workflowID))
			idStr := workflowID.String()
			slog.Debug(fmt.Sprintf("generated workflow ID String: %v", idStr))

			event := api.WorkflowStarted{
				ID:             api.WorkflowID(workflowID.String()),
				WorkflowFnName: data.WorkflowFnName,
				Input:          data.Input,
			}

			js, err := h.conn.JS()
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to get JetStream context: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					WorkflowID: "",
					Error:      "internal server error: failed to get JetStream context: " + err.Error()})

				slog.Debug(fmt.Sprintf("publishing error reply: %v", string(reply)))
				msg.Respond(reply)
				return
			}

			reply, err := h.conv.SerializeBinary(api.StartWorkflowReply{WorkflowID: workflowID.String()})
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to convert reply to bytes: %v", err))
				// Send a basic JSON error response
				errorReply := fmt.Sprintf(`{"error":"failed to serialize reply: %s","workflow_id":""}`, err.Error())
				slog.Debug(fmt.Sprintf("sending error reply: %s", errorReply))
				msg.Respond([]byte(errorReply))
				return
			}

			// Store input arguments in KV store using workflow function name as key
			kv, err := js.KeyValue(context.Background(), constant.WorkflowInputBucket)
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to get KV store: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					WorkflowID: "",
					Error:      "internal server error: failed to get KV store: " + err.Error()})
				msg.Respond(reply)
				return
			}

			inputArgsData, err := h.conv.SerializeBinary(data.Input)
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to serialize input args: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					WorkflowID: "",
					Error:      "internal server error: failed to serialize input args: " + err.Error()})
				msg.Respond(reply)
				return
			}

			_, err = kv.Put(context.Background(), data.WorkflowFnName, inputArgsData)
			if err != nil {
				slog.Debug(fmt.Sprintf("failed to store input args in KV: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					WorkflowID: "",
					Error:      "internal server error: failed to store input args: " + err.Error()})
				msg.Respond(reply)
				return
			}
			slog.Debug(fmt.Sprintf("stored input args in KV for workflow function: %s", data.WorkflowFnName))

			subject := fmt.Sprintf(constant.HistoryPublishSubjectPattern, idStr)
			eventBytes, _ := h.conv.SerializeBinary(event)
			_, err = js.Publish(context.Background(), subject, eventBytes)
			if err != nil {
				slog.Debug(fmt.Sprintf("error: %v", err))
				reply, _ := json.Marshal(api.StartWorkflowReply{
					Error: "internal server error: " + err.Error()})

				slog.Debug(fmt.Sprintf("publishing reply: %v", string(reply)))
				msg.Respond(reply)
				return
			}
			slog.Debug(fmt.Sprintf("published to history: %v", event))

			slog.Debug(fmt.Sprintf("FINAL REPLY CONTENT: %s", string(reply)))
			slog.Debug(fmt.Sprintf("REPLY LENGTH: %d", len(reply)))

			slog.Debug(fmt.Sprintf("publishing reply: %v", string(reply)))
			slog.Debug(fmt.Sprintf("SENDING RESPONSE TO REPLY SUBJECT: %s", msg.Reply))
			err = msg.Respond(reply)
			if err != nil {
				slog.Debug(fmt.Sprintf("Failed to send response: %v", err))
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
		constant.CommandRequestSubjectPattern,
		constant.ManagerCommandProcessorsConsumer,
		handler.HandleRequest,
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	<-ctx.Done()
	return nil
}
