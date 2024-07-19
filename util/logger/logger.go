/*
 * S370 - Wrapper for slog
 *
 * Copyright 2024, Richard Cornwell
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 */

package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type LogHandler struct {
	out   io.Writer
	h     slog.Handler
	mu    *sync.Mutex
	debug bool
}

func (h *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{h: h.h.WithAttrs(attrs), mu: h.mu}
}

func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{h: h.h.WithGroup(name), mu: h.mu}
}

func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"
	formattedTime := r.Time.Format("2006/01/02 15:04:05")

	strs := []string{formattedTime, level, r.Message}

	if r.NumAttrs() != 0 {
		r.Attrs(func(a slog.Attr) bool {
			strs = append(strs, a.Value.String())
			return true
		})
	}
	result := strings.Join(strs, " ") + "\n"
	b := []byte(result)

	h.mu.Lock()
	defer h.mu.Unlock()

	var err error
	if h.out != nil {
		_, err = h.out.Write(b)
	}

	if h.debug || r.Level > slog.LevelDebug {
		_, err = os.Stderr.Write(b)
	}
	return err
}

func (h *LogHandler) SetDebug(debug *bool) {
	h.debug = *debug
}

func NewHandler(file io.Writer, opts *slog.HandlerOptions, debug *bool) *LogHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &LogHandler{
		out: file,
		h: slog.NewTextHandler(file, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: nil,
		}),
		mu:    &sync.Mutex{},
		debug: *debug,
	}
}
