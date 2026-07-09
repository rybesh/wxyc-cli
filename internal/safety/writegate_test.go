package safety

import (
	"errors"
	"strings"
	"testing"
)

func TestGate_Authorize(t *testing.T) {
	tests := []struct {
		name       string
		allowWrite bool
		mutates    bool
		wantBlock  bool
	}{
		{name: "read command is always allowed when locked", allowWrite: false, mutates: false, wantBlock: false},
		{name: "read command is allowed when unlocked", allowWrite: true, mutates: false, wantBlock: false},
		{name: "write command is blocked when locked", allowWrite: false, mutates: true, wantBlock: true},
		{name: "write command is allowed when unlocked", allowWrite: true, mutates: true, wantBlock: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := Gate{AllowWrite: tt.allowWrite}
			err := g.Authorize("bin:add", tt.mutates)
			if tt.wantBlock {
				var be *BlockedError
				if !errors.As(err, &be) {
					t.Fatalf("Authorize() = %v, want *BlockedError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Authorize() = %v, want nil", err)
			}
		})
	}
}

func TestBlockedError_MentionsOpAndRemedy(t *testing.T) {
	err := (&BlockedError{Op: "flowsheet:add"}).Error()
	if !strings.Contains(err, "flowsheet:add") {
		t.Errorf("error %q should name the blocked op", err)
	}
	if !strings.Contains(err, "--write") || !strings.Contains(err, "WXYC_ALLOW_WRITE") {
		t.Errorf("error %q should tell the user how to unlock", err)
	}
}
