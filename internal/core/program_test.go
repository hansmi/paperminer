package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-chi/chi/v5"
	plclient "github.com/hansmi/paperhooks/pkg/client"
	"go.uber.org/zap/zaptest"
)

func TestNewProgram(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	app := kingpin.New("test", "")

	if _, err := NewProgram(ctx, zaptest.NewLogger(t), app); err != nil {
		t.Errorf("NewProgram() failed: %v", err)
	}

	for _, name := range []string{
		"cataloger_poll_interval",
		"listen_address",
		"object_default_owner_name",
		"paperless_server_timezone",
	} {
		if got := app.GetFlag(name); got == nil {
			t.Errorf("Missing flag %q", name)
		}
	}
}

func TestProgramRun(t *testing.T) {
	for _, tc := range []struct {
		name        string
		makeHandler func(context.CancelFunc) http.Handler
		check       func(*testing.T, error)
	}{
		{
			name: "http404",
			check: func(t *testing.T, err error) {
				var reqErr *plclient.RequestError

				if !errors.As(err, &reqErr) {
					t.Errorf("Run() failed: %v", err)
				} else if want := http.StatusNotFound; reqErr.StatusCode != want {
					t.Errorf("Run() failed with status code %d, want %d", reqErr.StatusCode, want)
				}
			},
		},
		{
			name: "cancel context",
			makeHandler: func(cancel context.CancelFunc) http.Handler {
				mux := chi.NewMux()
				mux.Get("/api/ui_settings/", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"user":{"id":123}}`)
				})
				mux.Get("/api/users/123/", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{}`)
				})
				mux.Get("/api/tags/", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{}`)
					cancel()
				})

				return mux
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Run() failed: %v", err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			app := kingpin.New("test", "")

			p, err := NewProgram(ctx, zaptest.NewLogger(t), app)
			if err != nil {
				t.Errorf("NewProgram() failed: %v", err)
			}

			p.name = app.Name
			p.storeDir = t.TempDir()

			handler := http.NotFoundHandler()

			if tc.makeHandler != nil {
				handler = tc.makeHandler(cancel)
			}

			ts := httptest.NewServer(handler)
			t.Cleanup(ts.Close)

			if _, err := app.Parse([]string{
				"--paperless_url", ts.URL,
			}); err != nil {
				t.Errorf("Parsing arguments failed: %v", err)
			}

			tc.check(t, p.Run(ctx))
		})
	}
}
