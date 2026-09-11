package models

import "testing"

func TestSentinelErrorsAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, err := range []error{ErrNotFound, ErrConflict, ErrForbidden, ErrInvalid} {
		if seen[err.Error()] {
			t.Fatalf("duplicate sentinel error: %q", err)
		}
		seen[err.Error()] = true
	}
}

func TestDomainConstraints(t *testing.T) {
	if StreamNameMaxLength != 128 {
		t.Fatalf("StreamNameMaxLength = %d, want 128", StreamNameMaxLength)
	}
	if HashtagNameMaxLength != 64 {
		t.Fatalf("HashtagNameMaxLength = %d, want 64", HashtagNameMaxLength)
	}
}
