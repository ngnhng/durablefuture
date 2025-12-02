package config

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/ngnhng/durablefuture/internal/server/types"
)

type LoggerConfig struct {
	Level          string      `env:"LOG_LEVEL"          envDefault:"info"`   // trace|debug|info|warn|error
	Format         string      `env:"LOG_FORMAT"         envDefault:"auto"`   // auto|json|text|pretty
	Output         string      `env:"LOG_OUTPUT"         envDefault:"stdout"` // stdout|stderr|file|multi
	FilePath       string      `env:"LOG_FILE_PATH"`                          // required if Output=file or includes file
	FileMode       os.FileMode `env:"LOG_FILE_MODE"      envDefault:"0644"`
	SampleRate     float64     `env:"LOG_SAMPLE_RATE"    envDefault:"1"`    // 0..1 (simple probabilistic sampling)
	ExtraFieldsRaw string      `env:"LOG_FIELDS"`                           // key1=val1,key2=val2
	OTELExporter   string      `env:"LOG_OTEL_EXPORTER"  envDefault:"none"` // none|otlp-http|otlp-grpc
	OTELEndpoint   string      `env:"LOG_OTEL_ENDPOINT"`
	Correlation    bool        `env:"LOG_CORRELATION"    envDefault:"true"` // TODO

	file    io.Writer
	fileMut sync.Mutex
}

// Writer returns the primary writer (first configured) to satisfy existing interface usage.
func (c *Config) Writer() io.Writer {
	writers := c.Writers()
	if len(writers) == 0 {
		return os.Stdout
	}
	return writers[0]
}

// Writers returns a slice of io.Writer for multi-output support.
// LOG_OUTPUT examples:
//
//	stdout
//	stderr
//	file (uses LOG_FILE_PATH)
//	file:/var/log/app.log
//	stdout,file             (comma-separated)
//	stdout,file:/tmp/app.log,stderr
//
// Unknown tokens are ignored with a warning.
func (c *Config) Writers() []io.Writer {
	outputs := strings.TrimSpace(c.Logger.Output)
	if outputs == "" {
		return []io.Writer{os.Stdout}
	}
	parts := strings.Split(outputs, ",")
	writers := make([]io.Writer, 0, len(parts))
	seen := make(map[string]struct{})

	addWriter := func(key string, w io.Writer) {
		if w == nil {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		writers = append(writers, w)
	}

	for _, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		lower := strings.ToLower(raw)
		// Support file:/path override syntax
		if strings.HasPrefix(lower, "file:") {
			path := strings.TrimPrefix(raw, "file:")
			w := c.openFile(path)
			addWriter("file:"+path, w)
			continue
		}
		switch lower {
		case "stdout":
			addWriter("stdout", os.Stdout)
		case "stderr":
			addWriter("stderr", os.Stderr)
		case "file":
			// Use configured FilePath
			if c.Logger.FilePath == "" {
				slog.Warn("LOG_OUTPUT includes 'file' but LOG_FILE_PATH not set; skipping")
				continue
			}
			w := c.openFile(c.Logger.FilePath)
			addWriter("file:"+c.Logger.FilePath, w)
		default:
			slog.Warn("unknown log output entry", "entry", raw)
		}
	}

	if len(writers) == 0 { // fallback
		return []io.Writer{os.Stdout}
	}
	return writers
}

// openFile opens or reuses a file writer.
func (c *Config) openFile(path string) io.Writer {
	if path == "" {
		return nil
	}
	// For simplicity reuse single file handle; extend to map if multiple distinct paths needed.
	if c.Logger.file != nil && c.Logger.FilePath == path {
		return c.Logger.file
	}
	c.Logger.fileMut.Lock()
	defer c.Logger.fileMut.Unlock()
	if c.Logger.file != nil && c.Logger.FilePath == path {
		return c.Logger.file
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, c.Logger.FileMode)
	if err != nil {
		slog.Warn("cannot open file for log output", "path", path, "error", err)
		return nil
	}
	c.Logger.FilePath = path
	c.Logger.file = f
	return f
}

// ParseExtraFields parses ExtraFieldsRaw into a map.
func (lc *LoggerConfig) ParseExtraFields() map[string]string {
	res := make(map[string]string)
	if lc == nil || lc.ExtraFieldsRaw == "" {
		return res
	}
	pairs := strings.Split(lc.ExtraFieldsRaw, ",")
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k != "" {
			res[k] = v
		}
	}
	return res
}

func (lc *LoggerConfig) ParseSampleRate() float64 {
	if lc == nil {
		return 1
	}
	r := lc.SampleRate
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

func (lc *LoggerConfig) ParseLevel() string {
	if lc == nil {
		return "info"
	}
	lvl := strings.ToLower(strings.TrimSpace(lc.Level))
	switch lvl {
	case "trace", "debug", "info", "warn", "error":
		return lvl
	default:
		return "info"
	}
}

// Interface compliance helpers for logger.LoggerOptions
func (c *Config) LogLevel() slog.Level {
	switch c.Logger.ParseLevel() {
	case "trace":
		return slog.Level(-8)
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *Config) LogFormat() string              { return c.Logger.Format }
func (c *Config) SampleRate() float64            { return c.Logger.ParseSampleRate() }
func (c *Config) OTELExporter() string           { return c.Logger.OTELExporter }
func (c *Config) OTELEndpoint() string           { return c.Logger.OTELEndpoint }
func (c *Config) ExtraFields() map[string]string { return c.Logger.ParseExtraFields() }
func (c *Config) ModeField() types.Mode          { return c.Mode }
