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
	"fmt"
	"strings"
	"time"
)

// NamespaceConfig provides configuration for multi-tenant namespace isolation
type NamespaceConfig struct {
	// Name of the namespace (must be DNS-safe)
	Name string `json:"name"`
	
	// Description of the namespace purpose
	Description string `json:"description"`
	
	// Retention policy for workflow data in this namespace
	RetentionPolicy *RetentionPolicy `json:"retention_policy,omitempty"`
	
	// Resource limits for this namespace
	ResourceLimits *ResourceLimits `json:"resource_limits,omitempty"`
}

// RetentionPolicy defines how long data should be kept
type RetentionPolicy struct {
	// WorkflowHistoryRetention - how long to keep workflow history
	WorkflowHistoryRetention time.Duration `json:"workflow_history_retention"`
	
	// CompletedWorkflowRetention - how long to keep completed workflow results
	CompletedWorkflowRetention time.Duration `json:"completed_workflow_retention"`
	
	// FailedWorkflowRetention - how long to keep failed workflow data for debugging
	FailedWorkflowRetention time.Duration `json:"failed_workflow_retention"`
}

// ResourceLimits defines resource constraints for a namespace
type ResourceLimits struct {
	// MaxConcurrentWorkflows - maximum number of concurrent workflows
	MaxConcurrentWorkflows int `json:"max_concurrent_workflows"`
	
	// MaxWorkflowsPerSecond - rate limit for new workflow executions
	MaxWorkflowsPerSecond int `json:"max_workflows_per_second"`
	
	// MaxActivityExecutionTime - maximum time an activity can run
	MaxActivityExecutionTime time.Duration `json:"max_activity_execution_time"`
}

// DefaultNamespaceConfig returns a default namespace configuration
func DefaultNamespaceConfig(name string) *NamespaceConfig {
	return &NamespaceConfig{
		Name:        name,
		Description: fmt.Sprintf("Default configuration for namespace: %s", name),
		RetentionPolicy: &RetentionPolicy{
			WorkflowHistoryRetention:   30 * 24 * time.Hour, // 30 days
			CompletedWorkflowRetention: 7 * 24 * time.Hour,  // 7 days
			FailedWorkflowRetention:    14 * 24 * time.Hour, // 14 days
		},
		ResourceLimits: &ResourceLimits{
			MaxConcurrentWorkflows:   1000,
			MaxWorkflowsPerSecond:    100,
			MaxActivityExecutionTime: 1 * time.Hour,
		},
	}
}

// ValidateNamespace checks if a namespace name is valid
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}
	
	if len(namespace) > 63 {
		return fmt.Errorf("namespace name too long (max 63 characters): %s", namespace)
	}
	
	if !isValidDNSName(namespace) {
		return fmt.Errorf("namespace name must be DNS-safe (lowercase letters, numbers, hyphens): %s", namespace)
	}
	
	return nil
}

// isValidDNSName checks if a string is a valid DNS name component
func isValidDNSName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// Must start and end with alphanumeric
	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return false
	}
	
	// Check each character
	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return false
		}
	}
	
	return true
}

// isAlphaNumeric checks if a byte is alphanumeric (lowercase)
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// GetNamespacedSubject returns a subject name scoped to a namespace
func GetNamespacedSubject(namespace, baseSubject string) string {
	if namespace == "" || namespace == "default" {
		return baseSubject
	}
	return fmt.Sprintf("%s.%s", namespace, baseSubject)
}

// GetNamespacedStreamName returns a stream name scoped to a namespace
func GetNamespacedStreamName(namespace, baseStreamName string) string {
	if namespace == "" || namespace == "default" {
		return baseStreamName
	}
	return fmt.Sprintf("%s_%s", strings.ToUpper(namespace), baseStreamName)
}

// GetNamespacedConsumerName returns a consumer name scoped to a namespace
func GetNamespacedConsumerName(namespace, baseConsumerName string) string {
	if namespace == "" || namespace == "default" {
		return baseConsumerName
	}
	return fmt.Sprintf("%s-%s", namespace, baseConsumerName)
}

// GetNamespacedKVBucket returns a KV bucket name scoped to a namespace
func GetNamespacedKVBucket(namespace, baseBucketName string) string {
	if namespace == "" || namespace == "default" {
		return baseBucketName
	}
	return fmt.Sprintf("%s-%s", namespace, baseBucketName)
}