package main

import (
    "io/ioutil"
    "strings"
    "testing"
)

func TestUpdateHosts_NewBlock(t *testing.T) {
    tmp := t.TempDir() + "/h"
    initial := "127.0.0.1 existing.local\n"
    _ = ioutil.WriteFile(tmp, []byte(initial), 0644)

    hosts := []string{"a.test", ""}
    if err := updateHosts(tmp, hosts); err != nil {
        t.Fatalf("updateHosts failed: %v", err)
    }
    out, _ := ioutil.ReadFile(tmp)
    s := string(out)
    if !strings.Contains(s, beginMarker) || !strings.Contains(s, "a.test") {
        t.Errorf("block not written: %s", s)
    }
}

func TestUpdateHosts_ReplaceBlock(t *testing.T) {
    tmp := t.TempDir() + "/h"
    initial := strings.Join([]string{
        "line1",
        beginMarker,
        "127.0.0.1 old",
        endMarker,
        "lineX",
    }, "\n") + "\n"
    _ = ioutil.WriteFile(tmp, []byte(initial), 0644)

    hosts := []string{"new.local"}
    if err := updateHosts(tmp, hosts); err != nil {
        t.Fatalf("updateHosts failed: %v", err)
    }
    out, _ := ioutil.ReadFile(tmp)
    s := string(out)
    if strings.Contains(s, "old") {
        t.Errorf("old entry not removed: %s", s)
    }
    if !strings.Contains(s, "new.local") {
        t.Errorf("new entry missing: %s", s)
    }
}
