package router

import (
	"encoding/json"
	"log"
	"os"

	"onql/config"
	"onql/engine"
)

var Registries map[string]any
var nats = engine.NatsClient{}

type Message struct {
	ID      string `json:"id"`      // who is sending (your keyword)
	RID     string `json:"rid"`     // unique request ID
	Target  string `json:"target"`  // which keyword to route to
	Payload string `json:"payload"` // arbitrary JSON payload
	Type    string `json:"type"`    // "request" or "response"
}

func init() {
	err := nats.Connect(config.Env("NATS_URL"), false)
	if err != nil {
		log.Printf("⚠️  could not connect to NATS: %v", err)
		return
	}

	registryPath := config.Env("EXTENSION_REGISTRY")
	Registries = make(map[string]any)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		log.Printf("⚠️  could not read extension registry %q: %v", registryPath, err)
		return
	}

	if err := json.Unmarshal(data, &Registries); err != nil {
		log.Printf("⚠️  could not parse extension registry JSON: %v", err)
		return
	}

	log.Printf("✅  loaded %d registry entries from %s", len(Registries), registryPath)
	setupExtensions()
}
