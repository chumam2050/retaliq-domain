package main

import (
    "flag"
    "log"
    "net/http"
    "os"
    "runtime"
    "strings"

)

func main() {
    // command-line flags
    var (
        cfgPath     string
        flagKey     string
        flagAllowed string
        flagPort    string
        saveConfig  bool
    )
    flag.StringVar(&cfgPath, "config", "", "path to config file")
    flag.StringVar(&flagKey, "apikey", "", "API key (overrides other sources)")
    flag.StringVar(&flagAllowed, "allowed", "", "comma-separated allowed IPs (overrides other sources)")
    flag.StringVar(&flagPort, "port", "", "port to listen on (overrides environment)")
    flag.BoolVar(&saveConfig, "save-config", false, "write effective configuration back to config file and exit")
    flag.Parse()

    // initial values
    var apiKey string
    var allowed map[string]struct{}
    allowed = make(map[string]struct{})

    // if a config file path wasn't provided, use default location
    if cfgPath == "" {
        cfgPath = DefaultConfigPath()
    }

    // if a config file exists attempt to load it
    if cfgPath != "" {
        if cfg, err := LoadConfig(cfgPath); err == nil {
            if cfg.APIKey != "" {
                apiKey = cfg.APIKey
            }
            if len(cfg.AllowedIPs) > 0 {
                allowed = parseAllowed(strings.Join(cfg.AllowedIPs, ","))
            }
        } else if !os.IsNotExist(err) {
            log.Fatalf("failed to read config file %s: %v", cfgPath, err)
        }
    }

    // command-line flags override everything
    if flagKey != "" {
        apiKey = flagKey
    }
    if flagAllowed != "" {
        allowed = parseAllowed(flagAllowed)
    }

    if apiKey == "" {
        log.Fatal("API key must be provided via -apikey or config file")
    }
    if len(allowed) == 0 {
        log.Fatal("allowed IP list must contain at least one address")
    }

    // if asked to save configuration, write file and exit
    if saveConfig {
        cfg := &Config{APIKey: apiKey}
        for ip := range allowed {
            cfg.AllowedIPs = append(cfg.AllowedIPs, ip)
        }
        if err := cfg.Save(cfgPath); err != nil {
            log.Fatalf("failed to save config: %v", err)
        }
        log.Printf("configuration written to %s", cfgPath)
        return
    }

    // determine port: command-line flag only; default to 8888
    port := flagPort
    if port == "" {
        port = "8888"
    }

    hostsPath := defaultHostsPath()
    log.Printf("starting domain helper on port %s, hosts file %s", port, hostsPath)

    handler := newHandler(apiKey, allowed, hostsPath)
    http.Handle("/hosts", handler)

    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}

func defaultHostsPath() string {
    if runtime.GOOS == "windows" {
        return `C:\Windows\System32\drivers\etc\hosts`
    }
    return "/etc/hosts"
}
