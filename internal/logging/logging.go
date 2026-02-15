package logging

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config controls how the structured logger is initialised.
type Config struct {
	FilePath    string
	Level       string
	MaxSizeMB   int
	MaxAgeDays  int
	MaxBackups  int
	ExtraWriter io.Writer // optional additional writer (e.g. SSE log hub)
}

func (c Config) withDefaults() Config {
	if c.MaxSizeMB == 0 {
		c.MaxSizeMB = 10
	}
	if c.MaxAgeDays == 0 {
		c.MaxAgeDays = 14
	}
	if c.MaxBackups == 0 {
		c.MaxBackups = 5
	}
	return c
}

// Writer holds the lumberjack logger and the stderr pipe used to capture
// Caddy output into the same rotated log file.
type Writer struct {
	lj         *lumberjack.Logger
	pipeWriter *os.File
	pipeReader *os.File
	origStderr *os.File
	done       chan struct{}
}

// Setup initialises structured JSON logging to a rotated file and redirects
// os.Stderr through a pipe so that Caddy access logs (written to stderr)
// also flow into the same file.
func Setup(cfg Config) (*Writer, error) {
	cfg = cfg.withDefaults()

	lj := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxBackups,
		Compress:   false,
	}

	// Set zerolog global logger to write JSON to lumberjack (and optional extra writer).
	var w io.Writer = lj
	if cfg.ExtraWriter != nil {
		w = zerolog.MultiLevelWriter(lj, cfg.ExtraWriter)
	}
	log.Logger = zerolog.New(w).Level(parseLevel(cfg.Level)).With().Timestamp().Logger()

	// Capture os.Stderr via os.Pipe so Caddy's stderr output reaches lumberjack.
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	origStderr := os.Stderr
	os.Stderr = pw

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(lj, pr)
		close(done)
	}()

	return &Writer{
		lj:         lj,
		pipeWriter: pw,
		pipeReader: pr,
		origStderr: origStderr,
		done:       done,
	}, nil
}

// Close drains the stderr pipe and closes the lumberjack logger.
func (w *Writer) Close() error {
	// Restore original stderr so later writes don't hit closed pipe.
	os.Stderr = w.origStderr

	// Close write end → io.Copy in goroutine sees EOF and drains.
	_ = w.pipeWriter.Close()
	<-w.done
	_ = w.pipeReader.Close()

	return w.lj.Close()
}

// parseLevel converts a level string to a zerolog.Level, defaulting to info.
func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
