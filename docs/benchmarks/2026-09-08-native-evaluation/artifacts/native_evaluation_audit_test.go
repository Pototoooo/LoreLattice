package service

import (
	"github.com/Pototoooo/lorelattice/internal/types"
	"testing"
)

func TestNativeEvaluationPassageConservation(t *testing.T) {
	ds := []*types.QAPair{{PIDs: []int{2, 3, 1, 4}, Passages: []string{"UNIX", "FreeBSD", "Mac", "DOS"}}}
	p := getPassageList(ds)
	if len(p) <= 4 || p[4] != "DOS" {
		t.Fatalf("highest PID 4 lost: len=%d passages=%q", len(p), p)
	}
}
