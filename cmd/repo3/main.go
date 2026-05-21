package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"repo3/internal/config"
	githubstore "repo3/internal/github"
	"repo3/internal/httpserver"
	"repo3/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: repo3 serve --addr :9000 --github-token $GITHUB_TOKEN --owner your-org")
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfg := config.Config{}
	fs.StringVar(&cfg.Addr, "addr", ":9000", "listen address")
	fs.StringVar(&cfg.GitHubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token")
	fs.StringVar(&cfg.Owner, "owner", "", "GitHub owner or organization")
	fs.StringVar(&cfg.DefaultBranch, "default-branch", "main", "default repository branch")
	fs.StringVar(&cfg.AccessKey, "access-key", "", "static access key for later auth support")
	fs.StringVar(&cfg.SecretKey, "secret-key", "", "static secret key for later auth support")
	fs.BoolVar(&cfg.AllowDeleteRepos, "allow-delete-repos", false, "allow deleting GitHub repositories")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var store storage.ObjectStore
	if cfg.GitHubToken != "" && cfg.Owner != "" {
		store = githubstore.NewStore(cfg.GitHubToken, cfg.Owner, cfg.DefaultBranch)
	} else {
		log.Print("github token or owner not provided; using in-memory storage")
		store = storage.NewMemoryStore()
	}
	srv := httpserver.New(cfg, store)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}
