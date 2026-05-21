package cli

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"repo3/internal/config"
	githubstore "repo3/internal/github"
	"repo3/internal/httpserver"
	"repo3/internal/storage"
)

func (c command) runServe(args []string) error {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		envFile = map[string]string{}
	}

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	cfg := config.Config{}
	fs.StringVar(&cfg.Addr, "addr", serveAddrDefault(envFile), "listen address")
	fs.StringVar(&cfg.GitHubToken, "github-token", envDefaultFromFile(envFile, "", "GITHUB_TOKEN"), "GitHub API token")
	fs.StringVar(&cfg.Owner, "owner", envDefaultFromFile(envFile, "", "REPO3_OWNER"), "GitHub owner or organization")
	fs.StringVar(&cfg.DefaultBranch, "default-branch", envDefaultFromFile(envFile, "main", "REPO3_DEFAULT_BRANCH", "GITHUB_DEFAULT_BRANCH"), "default repository branch")
	fs.StringVar(&cfg.AccessKey, "access-key", envDefaultFromFile(envFile, "", "REPO3_ACCESS_KEY"), "static access key for later auth support")
	fs.StringVar(&cfg.SecretKey, "secret-key", envDefaultFromFile(envFile, "", "REPO3_SECRET_KEY"), "static secret key for later auth support")
	fs.BoolVar(&cfg.AllowDeleteRepos, "allow-delete-repos", false, "allow deleting GitHub repositories")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var store storage.ObjectStore
	if cfg.GitHubToken != "" && cfg.Owner != "" {
		log.Printf("using GitHub backend owner=%s branch=%s addr=%s", cfg.Owner, cfg.DefaultBranch, cfg.Addr)
		store = githubstore.NewStore(cfg.GitHubToken, cfg.Owner, cfg.DefaultBranch)
	} else {
		log.Printf("github token or owner not provided; using in-memory storage addr=%s", cfg.Addr)
		store = storage.NewMemoryStore()
	}
	srv := httpserver.New(cfg, store)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}

func serveAddrDefault(envFile map[string]string) string {
	if addr := envDefaultFromFile(envFile, "", "REPO3_ADDR"); addr != "" {
		return addr
	}
	if port := envDefaultFromFile(envFile, "", "REPO3_PORT"); port != "" {
		if port[0] == ':' {
			return port
		}
		return ":" + port
	}
	return ":9000"
}

func envDefaultFromFile(envFile map[string]string, fallback string, keys ...string) string {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		if value := envFile[key]; value != "" {
			return value
		}
	}
	return fallback
}
