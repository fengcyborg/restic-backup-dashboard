package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/collector"
	"github.com/fengcyborg/restic-backup-dashboard/internal/config"
	"github.com/fengcyborg/restic-backup-dashboard/internal/server"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		printUsage(os.Stderr)
		return errors.New("a command is required")
	}
	switch arguments[0] {
	case "serve":
		return serve(arguments[1:])
	case "collect":
		return collect(arguments[1:])
	case "validate-config":
		return validateConfig(arguments[1:])
	case "healthcheck":
		return healthcheck(arguments[1:])
	case "version":
		fmt.Printf("restic-backup-dashboard %s (commit %s, built %s)\n", version, commit, date)
		return nil
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func serve(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	listen := flags.String("listen", "127.0.0.1:8080", "HTTP listen address")
	statusFile := flags.String("status-file", "/var/lib/restic-backup-dashboard/status.json", "sanitized status JSON path")
	demo := flags.Bool("demo", false, "serve embedded demonstration data")
	maxAge := flags.Duration("max-status-age", 3*time.Minute, "readiness freshness threshold")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("serve does not accept positional arguments")
	}
	handler, err := server.New(server.Options{
		StatusFile:   *statusFile,
		Demo:         *demo,
		MaxStatusAge: *maxAge,
	})
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() {
		log.Printf("dashboard listening on %s (demo=%t)", *listen, *demo)
		serverError <- httpServer.ListenAndServe()
	}()
	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	}
}

func collect(arguments []string) error {
	flags := flag.NewFlagSet("collect", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "/etc/restic-backup-dashboard/config.json", "collector configuration path")
	output := flags.String("output", "", "override status output path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("collect does not accept positional arguments")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	context, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	status := collector.New(cfg).Collect(context)
	outputPath := cfg.OutputPath(*output)
	if err := collector.WriteStatus(outputPath, status); err != nil {
		return err
	}
	fmt.Printf("wrote sanitized status to %s\n", outputPath)
	return nil
}

func validateConfig(arguments []string) error {
	flags := flag.NewFlagSet("validate-config", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "/etc/restic-backup-dashboard/config.json", "collector configuration path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("validate-config does not accept positional arguments")
	}
	if _, err := config.Load(*configPath); err != nil {
		return err
	}
	fmt.Println("configuration is valid")
	return nil
}

func healthcheck(arguments []string) error {
	flags := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	endpoint := flags.String("url", "http://127.0.0.1:8080/healthz", "health endpoint")
	timeout := flags.Duration("timeout", 5*time.Second, "request timeout")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	parsed, err := url.Parse(*endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("healthcheck URL must be an absolute HTTP(S) URL")
	}
	client := &http.Client{Timeout: *timeout}
	request, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", response.Status)
	}
	return nil
}

func printUsage(writer io.Writer) {
	lines := []string{
		"Restic Backup Dashboard",
		"",
		"Usage:",
		"  restic-backup-dashboard serve [flags]",
		"  restic-backup-dashboard collect [flags]",
		"  restic-backup-dashboard validate-config [flags]",
		"  restic-backup-dashboard healthcheck [flags]",
		"  restic-backup-dashboard version",
		"",
		"The collector runs on the backup host and writes sanitized JSON. The web",
		"server only reads that JSON and never needs backup credentials.",
	}
	_, _ = fmt.Fprintln(writer, strings.Join(lines, "\n"))
}
