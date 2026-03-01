package services

import (
	"context"
	"log/slog"
)

// emitFunc is set during startup to the Wails runtime.EventsEmit function.
var emitFunc func(ctx context.Context, eventName string, data ...interface{})

// SetEmitFunc sets the global Wails event emit function.
func SetEmitFunc(fn func(ctx context.Context, eventName string, data ...interface{})) {
	emitFunc = fn
}

// emitEvent emits a Wails event to the Svelte frontend.
func emitEvent(ctx context.Context, eventName string, data ...interface{}) {
	if emitFunc == nil {
		slog.Debug("emit skipped (no emit func)", "event", eventName)
		return
	}
	emitFunc(ctx, eventName, data...)
}
