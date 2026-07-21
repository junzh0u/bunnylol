package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

type RegexRoute struct {
	Pattern  string `json:"pattern"`
	URL      string `json:"url"`
	compiled *regexp.Regexp
}

type Config struct {
	Default  string            `json:"Default"`
	Keywords map[string]string `json:"Keywords"`
	Regexes  []RegexRoute      `json:"Regexes"`
}

func compileRegexes(config *Config) error {
	for i := range config.Regexes {
		re, err := regexp.Compile(config.Regexes[i].Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", config.Regexes[i].Pattern, err)
		}
		config.Regexes[i].compiled = re
	}
	return nil
}

func loadConfig(path string) (*Config, error) {
	var config Config
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if config.Default == "" {
		return nil, fmt.Errorf("no default site in config file")
	}

	if err := compileRegexes(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func watchConfig(path string, configPtr *atomic.Pointer[Config]) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastMod time.Time
	if info, err := os.Stat(path); err == nil {
		lastMod = info.ModTime()
	}
	slog.Info("Config watcher started", "path", path)

	for range ticker.C {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !info.ModTime().After(lastMod) {
			continue
		}
		slog.Info("Config change detected", "path", path)
		lastMod = info.ModTime()

		config, err := loadConfig(path)
		if err != nil {
			slog.Warn("Config reload failed, keeping old config", "error", err)
			continue
		}
		configPtr.Store(config)
		slog.Info("Config reloaded", "path", path)
	}
}

func siteRoot(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

func matchEngine(q string, config *Config) (string, string) {
	if q == "" {
		return siteRoot(config.Default), ""
	}

	tokens := strings.SplitN(q, " ", 2)
	key := strings.ToLower(tokens[0])
	if engine, ok := config.Keywords[key]; ok {
		if len(tokens) > 1 {
			slog.Info("HIT", "key", key, "engine", engine, "q", tokens[1])
			return engine, tokens[1]
		}
		root := siteRoot(engine)
		slog.Info("HIT_ROOT", "key", key, "root", root)
		return root, ""
	}
	for _, route := range config.Regexes {
		if route.compiled.MatchString(q) {
			slog.Info("MATCH", "regex", route.Pattern, "q", q)
			return route.URL, q
		}
	}
	return config.Default, q
}

func makeHandler(configPtr *atomic.Pointer[Config]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := configPtr.Load()
		q := r.URL.Query().Get("q")
		slog.Info("GET", "q", q)
		engine, query := matchEngine(q, config)
		dest := strings.ReplaceAll(engine, "%s", url.QueryEscape(query))
		http.Redirect(w, r, dest, http.StatusTemporaryRedirect)
	}
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	defaultConfig := "./config.json"
	if env := os.Getenv("BUNNYLOL_CONFIG"); env != "" {
		defaultConfig = env
	}
	configPath := flag.String("config", defaultConfig, "Config file path (env: BUNNYLOL_CONFIG)")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("Config loaded", "path", *configPath)

	var configPtr atomic.Pointer[Config]
	configPtr.Store(config)
	go watchConfig(*configPath, &configPtr)

	http.HandleFunc("/", makeHandler(&configPtr))

	slog.Info("Server started", "port", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	slog.Error("Server stopped", "error", err)
}
