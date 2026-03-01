package main

import (
    "bytes"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestHandler_Success(t *testing.T) {
    tmp := t.TempDir() + "/hosts"
    _ = ioutil.WriteFile(tmp, []byte("foo\n"), 0644)

    allowed := map[string]struct{}{"127.0.0.1":{}}
    h := newHandler("secret", allowed, tmp)

    body := []byte(`["a.test","b.test"]`)
    req := httptest.NewRequest("POST", "/hosts", bytes.NewReader(body))
    req.Header.Set("X-Api-Key", "secret")
    req.RemoteAddr = "127.0.0.1:1234"
    w := httptest.NewRecorder()

    h(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d body %s", w.Code, w.Body.String())
    }

    out, err := ioutil.ReadFile(tmp)
    if err != nil {
        t.Fatalf("read result: %v", err)
    }
    s := string(out)
    if !strings.Contains(s, "a.test") || !strings.Contains(s, "b.test") {
        t.Errorf("hosts not written: %s", s)
    }
}

func TestHandler_Unauthorized(t *testing.T) {
    h := newHandler("secret", map[string]struct{}{"1.2.3.4":{}}, "whatever")
    req := httptest.NewRequest("POST", "/hosts", nil)
    req.Header.Set("X-Api-Key", "wrong")
    req.RemoteAddr = "1.2.3.4:80"
    w := httptest.NewRecorder()
    h(w, req)
    if w.Code != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", w.Code)
    }
}

func TestHandler_Forbidden(t *testing.T) {
    h := newHandler("secret", map[string]struct{}{"9.9.9.9":{}}, "whatever")
    req := httptest.NewRequest("POST", "/hosts", nil)
    req.Header.Set("X-Api-Key", "secret")
    req.RemoteAddr = "1.2.3.4:80"
    w := httptest.NewRecorder()
    h(w, req)
    if w.Code != http.StatusForbidden {
        t.Errorf("expected 403, got %d", w.Code)
    }
}

func TestHandler_BadJSON(t *testing.T) {
    allowed := map[string]struct{}{"127.0.0.1":{}}
    h := newHandler("secret", allowed, "whatever")
    req := httptest.NewRequest("POST", "/hosts", bytes.NewReader([]byte("not json")))
    req.Header.Set("X-Api-Key", "secret")
    req.RemoteAddr = "127.0.0.1:1"
    w := httptest.NewRecorder()
    h(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
