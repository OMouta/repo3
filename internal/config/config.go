package config

type Config struct {
	Addr             string
	GitHubToken      string
	Owner            string
	DefaultBranch    string
	AccessKey        string
	SecretKey        string
	AllowDeleteRepos bool
}
