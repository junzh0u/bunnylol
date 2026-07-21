package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestMatchEngine(t *testing.T) {
	config := Config{
		Default: "https://google.com/search?q=%s",
		Keywords: map[string]string{
			"yt": "https://youtube.com/results?search_query=%s",
		},
		Regexes: []RegexRoute{
			{Pattern: `^\d{4}$`, URL: "https://example.com/year?q=%s"},
		},
	}
	compileRegexes(&config)

	tests := []struct {
		query      string
		wantEngine string
		wantQuery  string
	}{
		{"yt cats", "https://youtube.com/results?search_query=%s", "cats"},
		{"yt", "https://youtube.com", ""},
		{"2024", "https://example.com/year?q=%s", "2024"},
		{"hello world", "https://google.com/search?q=%s", "hello world"},
		{"", "https://google.com", ""},
	}

	for _, tt := range tests {
		engine, query := matchEngine(tt.query, &config)
		if engine != tt.wantEngine || query != tt.wantQuery {
			t.Errorf("matchEngine(%q) = (%q, %q), want (%q, %q)", tt.query, engine, query, tt.wantEngine, tt.wantQuery)
		}
	}
}

func TestHandlerRedirects(t *testing.T) {
	config := Config{
		Default: "https://google.com/search?q=%s",
		Keywords: map[string]string{
			"ddg": "https://duckduckgo.com/?q=%s",
		},
		Regexes: []RegexRoute{},
	}
	compileRegexes(&config)

	var configPtr atomic.Pointer[Config]
	configPtr.Store(&config)
	handler := makeHandler(&configPtr)

	tests := []struct {
		query      string
		wantPrefix string
	}{
		{"ddg golang", "https://duckduckgo.com/?q=golang"},
		{"ddg", "https://duckduckgo.com"},
		{"something else", "https://google.com/search?q=something+else"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/?q="+url.QueryEscape(tt.query), nil)
		w := httptest.NewRecorder()

		handler(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusTemporaryRedirect {
			t.Errorf("Expected status 307, got %d", res.StatusCode)
		}

		loc, err := res.Location()
		if err != nil {
			t.Fatal("Redirect location missing")
		}
		if loc.String() != tt.wantPrefix {
			t.Errorf("Redirect URL = %q; want %q", loc.String(), tt.wantPrefix)
		}
	}
}

func TestCompileRegexes(t *testing.T) {
	config := Config{
		Regexes: []RegexRoute{
			{Pattern: `^\d+$`, URL: "https://example.com/num?q=%s"},
			{Pattern: `^go\d+$`, URL: "https://golang.org/search?q=%s"},
		},
	}
	compileRegexes(&config)

	for _, route := range config.Regexes {
		if route.compiled == nil {
			t.Errorf("compiled regex for %q is nil", route.Pattern)
		}
		if route.compiled.String() != route.Pattern {
			t.Errorf("compiled regex string = %q, want %q", route.compiled.String(), route.Pattern)
		}
	}

	// Verify the compiled regexes actually match correctly
	if !config.Regexes[0].compiled.MatchString("1234") {
		t.Error(`expected ^\d+$ to match "1234"`)
	}
	if config.Regexes[0].compiled.MatchString("abc") {
		t.Error(`expected ^\d+$ not to match "abc"`)
	}
}

func TestCompileRegexesEmpty(t *testing.T) {
	config := Config{
		Regexes: []RegexRoute{},
	}
	compileRegexes(&config)

	if len(config.Regexes) != 0 {
		t.Fatalf("expected 0 regexes, got %d", len(config.Regexes))
	}
}

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	configContent := `{
		"Default": "https://google.com/search?q=%s",
		"Keywords": {
			"gh": "https://github.com/search?q=%s"
		},
		"Regexes": [
			{"pattern": "^go[0-9]+$", "url": "https://golang.org/search?q=%s"}
		]
	}`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if config.Default != "https://google.com/search?q=%s" {
		t.Errorf("expected default to be google, got %s", config.Default)
	}
	if config.Keywords["gh"] != "https://github.com/search?q=%s" {
		t.Errorf("expected keyword mapping for gh")
	}
	if len(config.Regexes) != 1 || config.Regexes[0].Pattern != "^go[0-9]+$" {
		t.Errorf("expected regex mapping for ^go[0-9]+$")
	}
	if config.Regexes[0].compiled == nil {
		t.Error("expected compiled regex for ^go[0-9]+$")
	}
}

func TestLoadConfigErrors(t *testing.T) {
	// Missing file
	_, err := loadConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error for missing file")
	}

	// Invalid JSON
	tempDir := t.TempDir()
	badPath := filepath.Join(tempDir, "bad.json")
	os.WriteFile(badPath, []byte("{invalid"), 0644)
	_, err = loadConfig(badPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Missing default
	noDefaultPath := filepath.Join(tempDir, "nodefault.json")
	os.WriteFile(noDefaultPath, []byte(`{"Keywords":{}}`), 0644)
	_, err = loadConfig(noDefaultPath)
	if err == nil {
		t.Error("expected error for missing default")
	}
}
