// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atif

import (
	"encoding/json"
	"fmt"
)

// Trajectory represents the root-level metadata and steps of an agent interaction.
// It implements the Agent Trajectory Interchange Format (ATIF).
type Trajectory struct {
	SchemaVersion          string         `json:"schema_version"`
	SessionID              string         `json:"session_id,omitempty"`
	TrajectoryID           string         `json:"trajectory_id,omitempty"`
	Agent                  Agent          `json:"agent"`
	Steps                  []Step         `json:"steps"`
	Notes                  string         `json:"notes,omitempty"`
	FinalMetrics           *FinalMetrics  `json:"final_metrics,omitempty"`
	ContinuedTrajectoryRef string         `json:"continued_trajectory_ref,omitempty"`
	Extra                  map[string]any `json:"extra,omitempty"`
	SubagentTrajectories   []Trajectory   `json:"subagent_trajectories,omitempty"`
}

// Agent specifies the agent configuration.
type Agent struct {
	Name            string         `json:"name"`
	Version         string         `json:"version"`
	ModelName       string         `json:"model_name,omitempty"`
	ToolDefinitions []any          `json:"tool_definitions,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

// FinalMetrics provides aggregate statistics for the entire trajectory.
type FinalMetrics struct {
	TotalPromptTokens     int            `json:"total_prompt_tokens,omitempty"`
	TotalCompletionTokens int            `json:"total_completion_tokens,omitempty"`
	TotalCachedTokens     int            `json:"total_cached_tokens,omitempty"`
	TotalCostUSD          float64        `json:"total_cost_usd,omitempty"`
	TotalSteps            int            `json:"total_steps,omitempty"`
	Extra                 map[string]any `json:"extra,omitempty"`
}

// Step represents a system prompt, a user message, or a complete agent turn.
type Step struct {
	StepID           int            `json:"step_id"`
	Timestamp        string         `json:"timestamp,omitempty"`
	Source           string         `json:"source"`
	ModelName        string         `json:"model_name,omitempty"`
	ReasoningEffort  any            `json:"reasoning_effort,omitempty"` // string or float64
	Message          MessageContent `json:"message"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	Observation      *Observation   `json:"observation,omitempty"`
	Metrics          *Metrics       `json:"metrics,omitempty"`
	Extra            map[string]any `json:"extra,omitempty"`
	LLMCallCount     *int           `json:"llm_call_count,omitempty"`
	IsCopiedContext  *bool          `json:"is_copied_context,omitempty"`
}

// MessageContent handles the message or observation content, which can be a
// string or an array of [ContentPart].
//
// We use distinct fields instead of an 'any' union type to provide a
// strongly-typed, auto-complete friendly API. UnmarshalJSON guarantees exactly
// one of these fields will be populated based on the raw JSON shape.
type MessageContent struct {
	Text  string
	Parts []ContentPart
}

func (m *MessageContent) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &m.Text)
	}
	if data[0] == '[' {
		return json.Unmarshal(data, &m.Parts)
	}
	return fmt.Errorf("MessageContent must be a string or array of ContentPart, got: %s", string(data))
}

func (m MessageContent) MarshalJSON() ([]byte, error) {
	if len(m.Parts) > 0 {
		return json.Marshal(m.Parts)
	}
	return json.Marshal(m.Text)
}

// ContentPart represents a part of a multimodal content.
type ContentPart struct {
	Type   string       `json:"type"` // "text" or "image"
	Text   string       `json:"text,omitempty"`
	Source *ImageSource `json:"source,omitempty"`
}

// ImageSource represents an image file referenced in the trajectory.
type ImageSource struct {
	MediaType string `json:"media_type"`
	Path      string `json:"path"`
}

// ToolCall represents a structured object for an agent's action.
type ToolCall struct {
	ToolCallID   string         `json:"tool_call_id"`
	FunctionName string         `json:"function_name"`
	Arguments    map[string]any `json:"arguments"`
	Extra        map[string]any `json:"extra,omitempty"`
}

// Metrics contains LLM operational and confidence data for a step.
type Metrics struct {
	PromptTokens       int            `json:"prompt_tokens,omitempty"`
	CompletionTokens   int            `json:"completion_tokens,omitempty"`
	CachedTokens       int            `json:"cached_tokens,omitempty"`
	CostUSD            float64        `json:"cost_usd,omitempty"`
	PromptTokenIDs     []int          `json:"prompt_token_ids,omitempty"`
	CompletionTokenIDs []int          `json:"completion_token_ids,omitempty"`
	Logprobs           []float64      `json:"logprobs,omitempty"`
	Extra              map[string]any `json:"extra,omitempty"`
}

// Observation records results from the environment or system events.
type Observation struct {
	Results []ObservationResult `json:"results"`
}

// ObservationResult represents the feedback from a single tool call or action.
type ObservationResult struct {
	SourceCallID          string                  `json:"source_call_id,omitempty"`
	Content               MessageContent          `json:"content"`
	SubagentTrajectoryRef []SubagentTrajectoryRef `json:"subagent_trajectory_ref,omitempty"`
	Extra                 map[string]any          `json:"extra,omitempty"`
}

// SubagentTrajectoryRef references a complete subagent trajectory.
type SubagentTrajectoryRef struct {
	TrajectoryID   string         `json:"trajectory_id,omitempty"`
	TrajectoryPath string         `json:"trajectory_path,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	Extra          map[string]any `json:"extra,omitempty"`
}
