package logger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

type LoggerConfig struct {
	Level     string // e.g., "info", "debug", "error"
	Output    string // "stdout", "file", or "remote"
	FilePath  string // used if Output is "file"
	RemoteURL string // used if Output is "remote"
}

// NewLogger: creates and configures a new logrus.Logger based on LoggerConfig.
func NewLogger(cfg LoggerConfig) (*logrus.Logger, error) {
	logger := logrus.New()

	// Parse and set the log level.
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %v", err)
	}
	logger.SetLevel(level)

	// Determine the output destination.
	var output io.Writer
	switch cfg.Output {
	case "stdout":
		output = os.Stdout
	case "file":
		f, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}
		output = f
	case "remote":
		remoteWriter, err := NewRemoteWriter(cfg.RemoteURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create remote writer: %v", err)
		}
		output = remoteWriter
	default:
		// Fallback to stdout if no valid output specified.
		output = os.Stdout
	}
	logger.SetOutput(output)

	// Optional: configure a formatter (e.g., JSONFormatter for structured logs).
	logger.SetFormatter(&logrus.JSONFormatter{})

	return logger, nil
}

// RemoteWriter implements io.Writer and sends logs to a remote endpoint.
type RemoteWriter struct {
	url string
}

// NewRemoteWriter creates a new RemoteWriter.
func NewRemoteWriter(url string) (*RemoteWriter, error) {
	// You might add URL validation here.
	return &RemoteWriter{url: url}, nil
}

// Write sends the log message to the remote URL using HTTP POST.
func (r *RemoteWriter) Write(p []byte) (n int, err error) {
	resp, err := http.Post(r.url, "text/plain", bytes.NewReader(p))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return 0, fmt.Errorf("failed to send log, status code %d", resp.StatusCode)
	}
	return len(p), nil
}
