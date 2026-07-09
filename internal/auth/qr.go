package auth

import (
	"io"

	"github.com/mdp/qrterminal/v3"
)

// renderQR draws a scannable QR of the verification URL to w using half-block
// characters (compact enough for a terminal). The WXYC iOS app reads the
// user_code out of this payload. Best-effort: a rendering failure is silently
// skipped since the plain-text code and URL are already shown.
func renderQR(w io.Writer, url string) {
	if url == "" {
		return
	}
	qrterminal.GenerateHalfBlock(url, qrterminal.L, w)
}
