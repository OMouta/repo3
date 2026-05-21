package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"repo3/internal/client"
)

type command struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type globalOptions struct {
	endpoint  string
	accessKey string
	secretKey string
	json      bool
	verbose   bool
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := command{stdin: stdin, stdout: stdout, stderr: stderr}
	if len(args) == 0 {
		cmd.usage()
		return fmt.Errorf("missing command")
	}

	switch args[0] {
	case "serve":
		return cmd.runServe(args[1:])
	case "mb":
		return cmd.runMB(args[1:])
	case "put":
		return cmd.runPut(args[1:])
	case "get":
		return cmd.runGet(args[1:])
	case "ls":
		return cmd.runLS(args[1:])
	case "rm":
		return cmd.runRM(args[1:])
	default:
		cmd.usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c command) usage() {
	fmt.Fprintln(c.stderr, "usage: repo3 <serve|mb|put|get|ls|rm> [flags]")
}

func (c command) parseClientFlags(name string, args []string) (*flag.FlagSet, globalOptions, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	opts := defaultGlobalOptions()
	c.addClientFlags(fs, &opts)
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return nil, opts, err
	}
	return fs, opts, nil
}

func defaultGlobalOptions() globalOptions {
	return globalOptions{
		endpoint:  envDefault("REPO3_ENDPOINT", "http://localhost:9000"),
		accessKey: os.Getenv("REPO3_ACCESS_KEY"),
		secretKey: os.Getenv("REPO3_SECRET_KEY"),
	}
}

func (c command) addClientFlags(fs *flag.FlagSet, opts *globalOptions) {
	fs.StringVar(&opts.endpoint, "endpoint", opts.endpoint, "Repo3 server endpoint")
	fs.StringVar(&opts.accessKey, "access-key", opts.accessKey, "static access key")
	fs.StringVar(&opts.secretKey, "secret-key", opts.secretKey, "static secret key")
	fs.BoolVar(&opts.json, "json", false, "write JSON output")
	fs.BoolVar(&opts.verbose, "verbose", false, "write verbose output")
}

func newClient(opts globalOptions) *client.Client {
	return &client.Client{
		Endpoint:  opts.endpoint,
		AccessKey: opts.accessKey,
		SecretKey: opts.secretKey,
		HTTP:      http.DefaultClient,
	}
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func background() context.Context {
	return context.Background()
}

func reorderFlags(args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") || isBoolFlag(arg) {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}

func isBoolFlag(arg string) bool {
	switch strings.TrimLeft(arg, "-") {
	case "json", "verbose", "create-bucket", "no-create-bucket":
		return true
	default:
		return false
	}
}
