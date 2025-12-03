// Package config provides configuration management for the DA Orchestrator.
//
// Configuration is loaded from environment variables using the env package.
// All configuration values have sensible defaults for development use.
//
// Example usage:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("HTTP server will listen on %s\n", cfg.GetHTTPAddr())
package config
