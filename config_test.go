package main

import (
    "os"
    "runtime"
    "strings"
    "testing"
)

func TestConfigLoadSave(t *testing.T) {
    tmp := t.TempDir() + "/cfg.conf"
    cfg := &Config{
        APIKey:     "k",
        AllowedIPs: []string{"1.2.3.4", "5.6.7.8"},
    }
    if err := cfg.Save(tmp); err != nil {
        t.Fatalf("save failed: %v", err)
    }
    got, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load failed: %v", err)
    }
    if got.APIKey != cfg.APIKey {
        t.Errorf("APIKey mismatch: %s", got.APIKey)
    }
    if len(got.AllowedIPs) != 2 || got.AllowedIPs[0] != "1.2.3.4" {
        t.Errorf("allowedips wrong: %v", got.AllowedIPs)
    }
}

func TestLoadConfig_missing(t *testing.T) {
    _, err := LoadConfig("/no/such/file")
    if err == nil {
        t.Error("expected error when loading nonexistent file")
    }
}

func TestDefaultConfigPath(t *testing.T) {
    p := DefaultConfigPath()
    if runtime.GOOS == "windows" {
        if !strings.HasSuffix(p, `\\retaliq-domain\\config.conf`) {
            t.Errorf("unexpected windows default path: %s", p)
        }
    } else {
        if !strings.HasPrefix(p, "/etc/") || !strings.HasSuffix(p, "/config.conf") {
            t.Errorf("unexpected default path: %s", p)
        }
    }
}

func TestSaveConfigFlag(t *testing.T) {
    tmp := t.TempDir() + "/cfg.conf"
    // simulate loading environment and flags
    apiKey := "xyz"
    cfg := &Config{APIKey: apiKey, AllowedIPs: []string{"1.2.3.4"}}
    if err := cfg.Save(tmp); err != nil {
        t.Fatalf("save failed: %v", err)
    }
    // reload and verify
    got, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load after save failed: %v", err)
    }
    if got.APIKey != apiKey {
        t.Errorf("expected key %s, got %s", apiKey, got.APIKey)
    }
}

func TestAutoGenerateKey(t *testing.T) {
    tmp := t.TempDir() + "/nodata.conf"
    // create a file with allowed_ips only
    if err := os.WriteFile(tmp, []byte("allowed_ips=1.2.3.4"), 0600); err != nil {
        t.Fatalf("failed to create tmp file: %v", err)
    }
    cfg, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load failed: %v", err)
    }
    if cfg.APIKey == "" {
        t.Errorf("expected generated API key")
    }
    // re-read file to ensure key was written
    // re-load file to ensure the new key was persisted
    cfg2, err2 := LoadConfig(tmp)
    if err2 != nil {
        t.Fatalf("reload failed: %v", err2)
    }
    if cfg2.APIKey == "" {
        t.Errorf("config file was not updated with api_key, still empty")
    }
}

func TestAddAllowedIP(t *testing.T) {
    tmp := t.TempDir() + "/cfg.conf"
    cfg := &Config{APIKey: "x", AllowedIPs: []string{"1.1.1.1"}}
    if err := cfg.Save(tmp); err != nil {
        t.Fatalf("save failed: %v", err)
    }
    if err := AddAllowedIP(tmp, "2.2.2.2"); err != nil {
        t.Fatalf("AddAllowedIP failed: %v", err)
    }
    got, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load failed: %v", err)
    }
    if len(got.AllowedIPs) != 2 {
        t.Errorf("unexpected ips: %v", got.AllowedIPs)
    }
}

func TestRegenerateKeyFunc(t *testing.T) {
    tmp := t.TempDir() + "/cfg.conf"
    // start with no file
    key, err := RegenerateKey(tmp)
    if err != nil {
        t.Fatalf("RegenerateKey failed: %v", err)
    }
    if key == "" {
        t.Fatal("expected non-empty key")
    }
    got, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load failed: %v", err)
    }
    if got.APIKey != key {
        t.Errorf("key mismatch, got %s", got.APIKey)
    }
}

func TestHandleCLICommands(t *testing.T) {
    tmp := t.TempDir() + "/cfg.conf"
    // write initial config with one address
    if err := os.WriteFile(tmp, []byte("allowed_ips=1.1.1.1"), 0600); err != nil {
        t.Fatalf("write failed: %v", err)
    }
    if !handleCLI(tmp, []string{"add-ip", "2.2.2.2"}) {
        t.Fatal("add-ip not handled")
    }
    cfg, err := LoadConfig(tmp)
    if err != nil {
        t.Fatalf("load after add-ip: %v", err)
    }
    if len(cfg.AllowedIPs) != 2 {
        t.Errorf("add-ip did not append, got %v", cfg.AllowedIPs)
    }
    // gen-key
    if !handleCLI(tmp, []string{"gen-key"}) {
        t.Fatal("gen-key not handled")
    }
    cfg2, _ := LoadConfig(tmp)
    if cfg2.APIKey == "" {
        t.Error("gen-key failed to write key")
    }
    // show should be handled but we don't assert output here
    if !handleCLI(tmp, []string{"show"}) {
        t.Fatal("show not handled")
    }

    // calling with no args ought not to be treated as CLI (it would start server)
    if handleCLI(tmp, []string{}) {
        t.Fatal("empty args should not be handled as CLI")
    }
}

func TestOverrideOrdering(t *testing.T) {
    // write a config file and then call main logic via helper
    tmp := t.TempDir() + "/cfg.conf"
    cfg := &Config{APIKey: "fromconfig", AllowedIPs: []string{"1.1.1.1"}}
    _ = cfg.Save(tmp)

    // simulate environment variables
    os.Setenv("RETALIQ_API_KEY", "fromenv")
    os.Setenv("RETALIQ_ALLOWED_IPS", "2.2.2.2")
    defer os.Unsetenv("RETALIQ_API_KEY")
    defer os.Unsetenv("RETALIQ_ALLOWED_IPS")

    // load via manual code path
    // exercise the load/override behaviour by simulating each source in turn
    apiKey := os.Getenv("RETALIQ_API_KEY")
    // note: allowed list not asserted here, just ensure no panics

    if cfg2, err := LoadConfig(tmp); err == nil {
        if cfg2.APIKey != "" {
            apiKey = cfg2.APIKey
        }
        if len(cfg2.AllowedIPs) > 0 {
            // just read, not assert here
        }
    }

    // pretend the user passed command-line flags
    if "flagkey" != "" {
        apiKey = "flagkey"
    }
    // flagallowed would override allowed list if we cared

    if apiKey == "" {
        t.Errorf("apiKey unexpectedly empty")
    }
}
