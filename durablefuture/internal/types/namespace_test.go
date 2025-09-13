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

package types

import (
	"testing"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{
			name:      "valid namespace",
			namespace: "my-namespace",
			wantErr:   false,
		},
		{
			name:      "valid namespace with numbers",
			namespace: "namespace123",
			wantErr:   false,
		},
		{
			name:      "empty namespace",
			namespace: "",
			wantErr:   true,
		},
		{
			name:      "too long namespace",
			namespace: "this-is-a-very-long-namespace-name-that-exceeds-the-maximum-allowed-length-for-dns-names",
			wantErr:   true,
		},
		{
			name:      "namespace with uppercase",
			namespace: "My-Namespace",
			wantErr:   true,
		},
		{
			name:      "namespace starting with hyphen",
			namespace: "-invalid",
			wantErr:   true,
		},
		{
			name:      "namespace ending with hyphen",
			namespace: "invalid-",
			wantErr:   true,
		},
		{
			name:      "namespace with special characters",
			namespace: "my_namespace",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetNamespacedSubject(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		baseSubject string
		expected    string
	}{
		{
			name:        "default namespace",
			namespace:   "default",
			baseSubject: "workflow.123.tasks",
			expected:    "workflow.123.tasks",
		},
		{
			name:        "empty namespace",
			namespace:   "",
			baseSubject: "workflow.123.tasks",
			expected:    "workflow.123.tasks",
		},
		{
			name:        "custom namespace",
			namespace:   "production",
			baseSubject: "workflow.123.tasks",
			expected:    "production.workflow.123.tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespacedSubject(tt.namespace, tt.baseSubject)
			if result != tt.expected {
				t.Errorf("GetNamespacedSubject() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetNamespacedStreamName(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		baseStreamName string
		expected       string
	}{
		{
			name:           "default namespace",
			namespace:      "default",
			baseStreamName: "WORKFLOW_TASKS",
			expected:       "WORKFLOW_TASKS",
		},
		{
			name:           "custom namespace",
			namespace:      "production",
			baseStreamName: "WORKFLOW_TASKS",
			expected:       "PRODUCTION_WORKFLOW_TASKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespacedStreamName(tt.namespace, tt.baseStreamName)
			if result != tt.expected {
				t.Errorf("GetNamespacedStreamName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetNamespacedConsumerName(t *testing.T) {
	tests := []struct {
		name             string
		namespace        string
		baseConsumerName string
		expected         string
	}{
		{
			name:             "default namespace",
			namespace:        "default",
			baseConsumerName: "workflow-worker-pool",
			expected:         "workflow-worker-pool",
		},
		{
			name:             "custom namespace",
			namespace:        "staging",
			baseConsumerName: "workflow-worker-pool",
			expected:         "staging-workflow-worker-pool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespacedConsumerName(tt.namespace, tt.baseConsumerName)
			if result != tt.expected {
				t.Errorf("GetNamespacedConsumerName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetNamespacedKVBucket(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		baseBucketName string
		expected       string
	}{
		{
			name:           "default namespace",
			namespace:      "default",
			baseBucketName: "workflow-input",
			expected:       "workflow-input",
		},
		{
			name:           "custom namespace",
			namespace:      "testing",
			baseBucketName: "workflow-input",
			expected:       "testing-workflow-input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespacedKVBucket(tt.namespace, tt.baseBucketName)
			if result != tt.expected {
				t.Errorf("GetNamespacedKVBucket() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDefaultNamespaceConfig(t *testing.T) {
	config := DefaultNamespaceConfig("test-namespace")
	
	if config.Name != "test-namespace" {
		t.Errorf("Expected Name = 'test-namespace', got '%s'", config.Name)
	}
	
	if config.RetentionPolicy == nil {
		t.Error("Expected RetentionPolicy to be set")
	}
	
	if config.ResourceLimits == nil {
		t.Error("Expected ResourceLimits to be set")
	}
	
	if config.ResourceLimits.MaxConcurrentWorkflows != 1000 {
		t.Errorf("Expected MaxConcurrentWorkflows = 1000, got %d", config.ResourceLimits.MaxConcurrentWorkflows)
	}
}