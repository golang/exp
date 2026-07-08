// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package atif provides Go types and JSON unmarshaling logic for the Agent Trajectory
// Interchange Format (ATIF).
//
// ATIF is a unified standard for logging agent interactions, actions, observations,
// and metrics across different autonomous frameworks (e.g., OpenHands, SWE-agent, Harbor).
// It solves the problem of competing log formats by providing a standardized schema
// for storing a sequence of environment events.
//
// The core struct is the Trajectory, which encapsulates the root-level metadata
// and an array of sequential Steps. A Step may contain tool calls, multi-modal
// messages, and granular token usage metrics.
//
// For more details on the ATIF specification and schema definition, see the official
// [ATIF RFC] and the [trajectory format documentation].
//
// [ATIF RFC]: https://github.com/harbor-framework/harbor/blob/main/rfcs/0001-trajectory-format.md
// [trajectory format documentation]: https://www.harborframework.com/docs/agents/trajectory-format
package atif
