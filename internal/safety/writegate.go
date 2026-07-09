// Package safety implements the read-only-by-default write gate. It is the
// client-side half of the two-layer safety model: it refuses to send a
// mutating request unless the operator has explicitly unlocked writes. The
// server's role check (403) is the independent backstop.
package safety

import "fmt"

// BlockedError is returned when a mutating operation is attempted while the
// gate is locked. Callers map it to a distinct process exit code so scripts
// and agents can branch on "blocked by policy" versus other failures.
type BlockedError struct {
	// Op is the command identifier, e.g. "flowsheet:add".
	Op string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf(
		"%q mutates state and the CLI is read-only; re-run with --write (interactive) or WXYC_ALLOW_WRITE=1 (scripted) to permit",
		e.Op,
	)
}

// Gate decides whether a command may run. The zero value is locked
// (read-only), which is the intended default posture.
type Gate struct {
	// AllowWrite unlocks mutating commands. It is set from the --write flag
	// or the WXYC_ALLOW_WRITE=1 environment variable.
	AllowWrite bool
}

// Authorize permits any read (mutates == false) unconditionally, and permits a
// write only when the gate is unlocked. op is used only for the error message.
func (g Gate) Authorize(op string, mutates bool) error {
	if !mutates || g.AllowWrite {
		return nil
	}
	return &BlockedError{Op: op}
}
