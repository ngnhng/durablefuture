// Copyright 2025 Nguyen-Nhat Nguyen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"
	"durablefuture/internal/types"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	color "github.com/fatih/color"

	"sync"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Logger struct {
	Slogger *slog.Logger
	*sdklog.LoggerProvider
}

type LoggerOptions struct {
	// Mode specifies the application mode (debug/release)
	Mode types.Mode

	// Writer is the writer to write the logs to
	Writer io.Writer
}

func NewLogger(ctx context.Context, opts *LoggerOptions) (*Logger, error) {
	if opts.Writer == nil {
		return nil, fmt.Errorf("no log writer")
	}
	handlers := make([]slog.Handler, 0)
	var loggerFactory *sdklog.LoggerProvider
	if opts.Mode == types.ModeDebug {
		handlers = append(handlers, &DebugHandler{
			out: opts.Writer,
			mut: sync.Mutex{},
		})
	} else {
		res, err := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("durablefuture"),
				semconv.ServiceVersion("v0.1.0-alpha1"),
			),
		)
		logExporter, err := otlploghttp.New(ctx)
		if err != nil {
			return nil, err
		}

		logProcessor := sdklog.NewBatchProcessor(logExporter, nil)

		loggerFactory = sdklog.NewLoggerProvider(
			sdklog.WithProcessor(logProcessor),
			sdklog.WithResource(res),
		)

		otelHandler := otelslog.NewHandler(
			"otelHandler", otelslog.WithLoggerProvider(loggerFactory))

		handlers = append(handlers,
			otelHandler,
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))

	}

	return &Logger{
		Slogger:        slog.New(&MultiHandler{handlers}),
		LoggerProvider: loggerFactory,
	}, nil
}

type (
	DebugHandler struct {
		out   io.Writer
		attrs []slog.Attr
		mut   sync.Mutex
	}

	MultiHandler struct {
		handlers []slog.Handler
	}
)

var _ slog.Handler = (*DebugHandler)(nil)

// Handle implements slog.Handler
func (h *DebugHandler) Handle(_ context.Context, r slog.Record) error {
	h.mut.Lock()
	defer h.mut.Unlock()

	timeStr := color.New(color.FgHiBlack).Sprint(r.Time.Format("15:04:05"))
	level := levelColor(r.Level)
	attrs := append(h.attrs, []slog.Attr{}...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	logEntry := fmt.Sprintf("%s %s %s%s\n",
		timeStr,
		level,
		r.Message,
		formatAttributes(attrs),
	)

	_, err := h.out.Write([]byte(logEntry))
	return err
}

// WithAttrs implements slog.Handler
func (h *DebugHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DebugHandler{
		out:   h.out,
		attrs: append(h.attrs, attrs...),
		mut:   sync.Mutex{}, // A mutex must not be copied
	}
}

// WithGroup implements slog.Handler
func (h *DebugHandler) WithGroup(name string) slog.Handler {
	return h
}

// Enabled implements slog.Handler
func (h *DebugHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

// Enabled implements slog.Handler
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements slog.Handler
func (m *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		// Best-effort handling: we log an error if a handler fails but continue.
		if err := h.Handle(ctx, record); err != nil {
			slog.Error("error from slog handler", "error", err)
		}
	}
	return nil
}

// WithAttrs implements slog.Handler
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

// WithGroup implements slog.Handler
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

// levelColor returns a colored string representation of the log level.
func levelColor(level slog.Level) string {
	var bg, fg color.Attribute
	switch level {
	case slog.LevelDebug:
		bg, fg = color.BgMagenta, color.FgWhite
	case slog.LevelInfo:
		bg, fg = color.BgBlue, color.FgWhite
	case slog.LevelWarn:
		bg, fg = color.BgYellow, color.FgBlack
	case slog.LevelError:
		bg, fg = color.BgRed, color.FgWhite
	default:
		bg, fg = color.BgWhite, color.FgBlack
	}

	return color.New(bg, fg, color.Bold).Sprint(" " + strings.ToUpper(level.String()) + " ")
}

// formatAttributes formats a slice of attributes as a space-separated string.
func formatAttributes(attrs []slog.Attr) string {
	if len(attrs) == 0 {
		return ""
	}

	var parts []string
	for _, attr := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%s", attr.Key, formatAttrValue(attr.Value)))
	}

	return " " + strings.Join(parts, " ")
}

// formatAttrValue formats a slog.Value based on its kind.
func formatAttrValue(v slog.Value) string {
	if valuer, ok := v.Any().(slog.LogValuer); ok {
		return formatAttrValue(valuer.LogValue())
	}

	switch v.Kind() {
	case slog.KindString:
		return fmt.Sprintf("%q", v.String())
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%f", v.Float64())
	case slog.KindBool:
		return fmt.Sprintf("%t", v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	case slog.KindAny:
		if m, ok := v.Any().(map[string]string); ok {
			parts := make([]string, 0, len(m))
			for k, v := range m {
				parts = append(parts, fmt.Sprintf("%s:%s", k, v))
			}

			return strings.Join(parts, " ")
		}

		return fmt.Sprintf("%v", v.Any())
	default:
		return fmt.Sprintf("%v", v)
	}
}
