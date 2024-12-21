package config

import "gitlab.com/etke.cc/go/env"

const prefix = "suae" // Synapse User Auto Erase

// Config is a struct that holds the configuration for the application.
type Config struct {
	Host     string   // Synapse host, e.g. "https://matrix.your-server.com" (without trailing slash)
	Token    string   // Synapse homeserver admin token
	DryRun   bool     // If true, no user will be erased
	Redact   bool     // If true, all messages sent by the user will be redacted
	Prefixes []string // Prefixes to omit/ignore
	TTL      int      // Time to live in days. After that time the user will be erased
}

// New creates a new Config struct and reads the configuration from the environment.
func New() *Config {
	env.SetPrefix(prefix)

	return &Config{
		Host:     env.String("host"),
		Token:    env.String("token"),
		Prefixes: env.Slice("prefixes"),
		DryRun:   env.Bool("dryrun"),
		TTL:      env.Int("ttl"),
	}
}
