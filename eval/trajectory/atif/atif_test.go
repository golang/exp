// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atif_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/eval/trajectory/atif"
)

func TestRoundTrip(t *testing.T) {
	// A normal test for a format package like this is a "round-trip" test.
	// We read the JSON, unmarshal it into our struct, marshal it back to JSON,
	// and verify that the re-marshaled JSON contains all the same data as the original.
	pattern := filepath.Join("testdata", "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob trajectory files: %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("No trajectory files found in testdata.")
	}

	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Errorf("Failed to read trajectory file %s: %v", match, err)
			continue
		}

		// Unmarshal into our struct
		var traj1 atif.Trajectory
		if err := json.Unmarshal(data, &traj1); err != nil {
			t.Errorf("Failed to unmarshal trajectory %s: %v", match, err)
			continue
		}

		// Marshal it back to JSON
		remarshaled, err := json.Marshal(traj1)
		if err != nil {
			t.Errorf("Failed to re-marshal trajectory %s: %v", match, err)
			continue
		}

		// Unmarshal the re-marshaled JSON into a second struct
		var traj2 atif.Trajectory
		if err := json.Unmarshal(remarshaled, &traj2); err != nil {
			t.Fatalf("Failed to unmarshal remarshaled JSON: %v", err)
		}

		// Compare the two structs deeply to ensure no data was lost or corrupted
		if diff := cmp.Diff(traj1, traj2, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Round-trip mismatch for %s (-want +got):\n%s", filepath.Base(match), diff)
		}
	}
}
