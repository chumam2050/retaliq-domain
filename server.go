package main

import (
    "encoding/json"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"
)

const (
    beginMarker = "# BEGIN RETALIQHOSTS inline"
    endMarker   = "# END RETALIQHOSTS inline"
)

// parseAllowed converts a comma-separated list into a set of addresses.
func parseAllowed(raw string) map[string]struct{} {
    m := make(map[string]struct{})
    for _, part := range strings.Split(raw, ",") {
        if ip := strings.TrimSpace(part); ip != "" {
            m[ip] = struct{}{}
        }
    }
    return m
}

// newHandler returns an http.HandlerFunc enforcing apiKey and allowed list.
func newHandler(apiKey string, allowed map[string]struct{}, hostsPath string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        if r.URL.Path != "/hosts" {
            http.NotFound(w, r)
            return
        }

        key := r.Header.Get("X-Api-Key")
        if key != apiKey {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // verify remote address
        host, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            host = r.RemoteAddr
        }
        if _, ok := allowed[host]; !ok {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        body, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "read error", http.StatusBadRequest)
            return
        }
        defer r.Body.Close()

        var hosts []string
        if err := json.Unmarshal(body, &hosts); err != nil {
            http.Error(w, "invalid json", http.StatusBadRequest)
            return
        }

        if err := updateHosts(hostsPath, hosts); err != nil {
            log.Printf("failed to update hosts: %v", err)
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
    }
}

// updateHosts replaces the inline block in the given file with entries for hosts.
func updateHosts(path string, hosts []string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    lines := strings.Split(string(data), "\n")
    var out []string
    inBlock := false
    for _, l := range lines {
        switch l {
        case beginMarker:
            inBlock = true
            continue
        case endMarker:
            inBlock = false
            continue
        }
        if !inBlock {
            out = append(out, l)
        }
    }

    // build new block
    out = append(out, beginMarker)
    for _, h := range hosts {
        h = strings.TrimSpace(h)
        if h == "" {
            continue
        }
        out = append(out, "127.0.0.1 "+h)
        out = append(out, "::1 "+h)
    }
    out = append(out, endMarker)

    result := strings.Join(out, "\n")
    if !strings.HasSuffix(result, "\n") {
        result += "\n"
    }

    // preserve existing permissions if possible
    mode := os.FileMode(0644)
    if fi, err := os.Stat(path); err == nil {
        mode = fi.Mode()
    }

    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, []byte(result), mode); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
