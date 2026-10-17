// Command tracker is the Music Tracker backend: HTTP API, scheduler and workers
// in a single binary. This package is the composition root.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const usage = `Usage: tracker <command> [args]

Commands:
  serve            start HTTP API, scheduler and workers (default)
  migrate up       apply all pending migrations
  migrate down     roll back the last migration
  migrate version  print the current schema version
  hash-password    read a password from stdin and print APP_PASSWORD_HASH
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "serve":
		return serve(ctx)
	case "migrate":
		return migrateCmd(args)
	case "hash-password":
		return hashPassword(os.Stdin, os.Stdout)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}
