package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"k8s.io/client-go/rest"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/klog/v2"
)

// Initialize runs after --logging-format is applied, so the startup line uses the configured format.
func TestInitializeLogsStartup(t *testing.T) {
	var buf bytes.Buffer
	logger, _ := rfc3339JSONFactory{}.Create(logsapi.LoggingConfiguration{}, logsapi.LoggingOptions{ErrorStream: &buf})
	klog.SetLogger(logger)
	defer klog.ClearLogger()

	s := &yandexCloudDNSSolver{groupName: "acme.test"}
	if err := s.Initialize(&rest.Config{Host: "https://127.0.0.1:1"}, nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	klog.Flush()

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not a single JSON line: %v\noutput: %q", err, buf.String())
	}
	if entry["msg"] != "Starting cert-manager-webhook-yandex" || entry["groupName"] != "acme.test" {
		t.Errorf("startup line = %v", entry)
	}
}
