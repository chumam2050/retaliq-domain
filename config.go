package main

import (
    "encoding/json"
    "os"
    "runtime"
)

// Config holds persistent configuration for the domain helper service.
// It is stored in JSON format by default (path may be provided via -config).
// A simple file is chosen because it's cross-platform and easily editable by
// helper scripts; on Windows the Windows registry could be used instead but
// a text file works equally well and is easier to manage from non-PowerShell.
//
// Example:
// {
//     "api_key": "secret",
//     "allowed_ips": ["127.0.0.1", "172.17.0.1"]
// }

type Config struct {
    APIKey     string   `json:"api_key"`
    AllowedIPs []string `json:"allowed_ips"`
}

// DefaultConfigPath returns a sensible location for the configuration file
// on the current platform.  The caller is free to override this with the
// -config flag.
func DefaultConfigPath() string {
    if runtime.GOOS == "windows" {
        return os.Getenv("ProgramData") + "\\retaliq-domain\\config.json"
    }
    return "/etc/retaliq-domain.json"
}

// LoadConfig reads the JSON configuration from path. If the file does not
// exist or cannot be parsed an error is returned.
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

// Save writes the configuration to the given path atomically.
func (c *Config) Save(path string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0600); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
