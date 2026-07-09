package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rybesh/wxyc-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd(app *App) *cobra.Command {
	var device bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in and store a session token (password never stored)",
		Long: "Sign in and store a session token. By default prompts for your\n" +
			"password. With --device, uses QR/device-authorization: approve the\n" +
			"sign-in from the WXYC iOS app and no password is entered here.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var (
				strategy auth.SignInStrategy
				who      string
			)
			if device {
				strategy = auth.DeviceStrategy{
					AuthBase: app.cfg.AuthBase,
					HTTP:     &http.Client{Timeout: 30 * time.Second},
					ClientID: deviceClientID(),
					Out:      app.stderr,
				}
				who = "device"
			} else {
				ident, err := prompt("WXYC email or username: ")
				if err != nil {
					return err
				}
				password, err := promptSecret("Password: ")
				if err != nil {
					return err
				}
				strategy = auth.PasswordStrategy{
					AuthBase: app.cfg.AuthBase,
					HTTP:     &http.Client{Timeout: 30 * time.Second},
					Ident:    ident,
					Password: password,
				}
				who = ident
			}

			token, err := strategy.Login(cmd.Context())
			if err != nil {
				return err
			}
			if err := app.store.Save(app.cfg.Profile, token); err != nil {
				return fmt.Errorf("saving session: %w", err)
			}
			fmt.Fprintf(app.stdout, "signed in as %s (profile %q)\n", who, app.cfg.Profile)
			return nil
		},
	}
	cmd.Flags().BoolVar(&device, "device", false, "sign in via QR/device authorization (approve from the iOS app)")
	return cmd
}

// deviceClientID is the RFC 8628 client_id sent to /device/code. Overridable
// in case the deployment ever enables client validation.
func deviceClientID() string {
	if v := os.Getenv("WXYC_DEVICE_CLIENT_ID"); v != "" {
		return v
	}
	return "wxyc-cli"
}

func prompt(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	var s string
	if _, err := fmt.Scanln(&s); err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func promptSecret(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
