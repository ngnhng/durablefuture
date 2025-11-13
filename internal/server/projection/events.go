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

package projection

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/ngnhng/durablefuture/api"
	"github.com/ngnhng/durablefuture/api/serde"
)

var workflowEventFactories = map[string]func() api.WorkflowEvent{
	(&api.WorkflowStarted{}).EventName():   func() api.WorkflowEvent { return &api.WorkflowStarted{} },
	(&api.ActivityScheduled{}).EventName(): func() api.WorkflowEvent { return &api.ActivityScheduled{} },
	(&api.ActivityStarted{}).EventName():   func() api.WorkflowEvent { return &api.ActivityStarted{} },
	(&api.ActivityCompleted{}).EventName(): func() api.WorkflowEvent {
		return &api.ActivityCompleted{}
	},
	(&api.ActivityFailed{}).EventName(): func() api.WorkflowEvent { return &api.ActivityFailed{} },
	(&api.WorkflowFailed{}).EventName(): func() api.WorkflowEvent { return &api.WorkflowFailed{} },
	(&api.WorkflowCompleted{}).EventName(): func() api.WorkflowEvent {
		return &api.WorkflowCompleted{}
	},
}

func decodeWorkflowEvent(msg jetstream.Msg, conv serde.BinarySerde) (api.WorkflowEvent, error) {
	payload := msg.Data()
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty workflow event payload")
	}

	eventType := msg.Headers().Get(api.ChronicleEventNameHeader)
	if eventType == "" {
		var err error
		eventType, err = inferWorkflowEventType(payload, conv)
		if err != nil {
			return nil, err
		}
	}

	factory, ok := workflowEventFactories[eventType]
	if !ok {
		return nil, fmt.Errorf("unsupported workflow event type %q", eventType)
	}

	event := factory()
	// Use the configured serializer instead of hardcoded JSON
	if err := conv.DeserializeBinary(payload, event); err != nil {
		return nil, fmt.Errorf("decode workflow event (%s): %w", eventType, err)
	}

	return event, nil
}

func inferWorkflowEventType(payload []byte, conv serde.BinarySerde) (string, error) {
	// Deserialize to a generic map to inspect fields
	var raw map[string]any
	if err := conv.DeserializeBinary(payload, &raw); err != nil {
		return "", fmt.Errorf("infer workflow event type: %w", err)
	}

	if hasKey(raw, "wf_name") {
		switch {
		case hasKey(raw, "result"):
			return (&api.ActivityCompleted{}).EventName(), nil
		case hasKey(raw, "error"):
			return (&api.ActivityFailed{}).EventName(), nil
		case hasKey(raw, "input"):
			return (&api.ActivityScheduled{}).EventName(), nil
		default:
			return (&api.ActivityStarted{}).EventName(), nil
		}
	}

	switch {
	case hasKey(raw, "result"):
		return (&api.WorkflowCompleted{}).EventName(), nil
	case hasKey(raw, "error"):
		return (&api.WorkflowFailed{}).EventName(), nil
	case hasKey(raw, "input"):
		return (&api.WorkflowStarted{}).EventName(), nil
	default:
		return "", fmt.Errorf("could not infer workflow event type from payload")
	}
}

func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}
