package main

import (
    "encoding/base64"
    "os"
    "runtime"
    "strings"
    "crypto/rand"
)

// Config holds persistent configuration for the domain helper service.
// We use a plain key=value file, similar to Apache/INI snippets.  The
// format is intentionally minimal so it can be edited by hand or by the
// setup helper without needing a JSON parser.  Lines beginning with `#` are
// ignored as comments.
//
// Example:
// api_key = secret
// allowed_ips = 127.0.0.1,172.17.0.1

type Config struct {
    APIKey     string
    AllowedIPs []string
}

// DefaultConfigPath returns a sensible location for the configuration file
// on the current platform.  The caller is free to override this with the
// -config flag.
func DefaultConfigPath() string {
    if runtime.GOOS == "windows" {
        return os.Getenv("ProgramData") + "\\retaliq-domain\\config.conf"
    }
    return "/etc/retaliq-domain/config.conf"
}

// LoadConfig reads the key=value configuration from path. If the file does
// not exist or cannot be parsed an error is returned.
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    lines := strings.Split(string(data), "\n")
    cfg := &Config{}
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        switch key {
        case "api_key":
            cfg.APIKey = val
        case "allowed_ips":
            if val != "" {
                for _, ip := range strings.Split(val, ",") {
                    cfg.AllowedIPs = append(cfg.AllowedIPs, strings.TrimSpace(ip))
                }
            }
        }
    }

    // if there was no API key we generate one and persist it, so that subsequent
    // runs and the helper script can rely on its presence.
    if cfg.APIKey == "" {
        cfg.APIKey = generateKey()
        // attempt to save, ignore errors (file may be read-only)
        _ = cfg.Save(path)
    }

    return cfg, nil
}

// generateKey returns a URL-safe random string suitable for use
// as an API key.  The length is 32 bytes base64-encoded, similar to what
// the helper script produces with openssl.
func generateKey() string {
    b := make([]byte, 24)
    if _, err := rand.Read(b); err != nil {
        return ""
    }
    return base64.RawURLEncoding.EncodeToString(b)
}

// Save writes the configuration to the given path in key=value form.
// The write is performed atomically.
// AddAllowedIP loads the config from path, appends the given ip to the
// allowed list (if not already present), and saves the file.
func AddAllowedIP(path, ip string) error {
    cfg, err := LoadConfig(path)
    if err != nil && !os.IsNotExist(err) {
        return err
    }
    // ensure slice initialized
    if cfg == nil {
        cfg = &Config{}
    }
    for _, existing := range cfg.AllowedIPs {
        if existing == ip {
            return cfg.Save(path)
        }
    }
    cfg.AllowedIPs = append(cfg.AllowedIPs, ip)
    return cfg.Save(path)
}

// RegenerateKey generates a new random API key, writes it into the config file
// at path (creating the file if necessary) and returns the key.
func RegenerateKey(path string) (string, error) {
    cfg, err := LoadConfig(path)
    if err != nil && !os.IsNotExist(err) {
        return "", err
    }
    if cfg == nil {
        cfg = &Config{}
    }
    key := generateKey()
    cfg.APIKey = key
    if err := cfg.Save(path); err != nil {
        return "", err
    }
    return key, nil
}

func (c *Config) Save(path string) error {
    var b strings.Builder
    if c.APIKey != "" {
        b.WriteString("api_key = ")
        b.WriteString(c.APIKey)
        b.WriteString("\n")
    }
    if len(c.AllowedIPs) > 0 {
        b.WriteString("allowed_ips = ")
        b.WriteString(strings.Join(c.AllowedIPs, ","))
        b.WriteString("\n")
    }
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, []byte(b.String()), 0600); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
