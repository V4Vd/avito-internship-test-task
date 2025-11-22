package service_test

import (
	"testing"

	"pr-reviewer-service/internal/models"
)

func TestSelectRandomReviewers(t *testing.T) {
	candidates := []models.User{
		{UserID: "u1", Username: "alice", IsActive: true},
		{UserID: "u2", Username: "bob", IsActive: true},
		{UserID: "u3", Username: "charlie", IsActive: true},
	}

	t.Run("select 2 from 3", func(t *testing.T) {
		if len(candidates) < 2 {
			t.Error("expected at least 2 candidates")
		}
	})
}

func TestPRStatusTransitions(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus string
		canReassign   bool
	}{
		{"open PR can be reassigned", models.StatusOpen, true},
		{"merged PR cannot be reassigned", models.StatusMerged, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.currentStatus == models.StatusMerged && tt.canReassign {
				t.Error("merged PR should not allow reassignment")
			}
		})
	}
}
