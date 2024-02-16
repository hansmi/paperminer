package paperminercmd

import (
	"context"
	"fmt"
	stdlog "log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hansmi/paperminer/internal/core"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// Main implements the main function of the paperminer command.
func Main() {
	rand.Seed(time.Now().UnixNano())

	ls, err := core.NewLoggingSetup()
	if err != nil {
		stdlog.Fatalf("Initializing logger failed: %v", err)
	}

	defer ls.ReplaceGlobals()()

	customLogLevel := zap.InfoLevel

	app := kingpin.CommandLine
	app.DefaultEnvars()
	app.Flag("log_level", "Log level for stderr.").
		Default(customLogLevel.String()).
		SetValue(&customLogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer stop()

	p, err := core.NewProgram(ctx, ls.Logger(), app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		os.Exit(1)
	}

	kingpin.Parse()

	ls.SetLevel(customLogLevel)

	if err := p.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
