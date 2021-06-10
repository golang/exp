package eventtest

import (
	"context"

	"golang.org/x/exp/event"
)

type CaptureHandler struct {
	Got []event.Event
}

func (h *CaptureHandler) Event(ctx context.Context, ev *event.Event) context.Context {
	h.Got = append(h.Got, *ev)
	got := &h.Got[len(h.Got)-1]
	got.Labels = make([]event.Label, len(ev.Labels))
	copy(got.Labels, ev.Labels)
	return ctx
}

func (h *CaptureHandler) Reset() {
	if len(h.Got) > 0 {
		h.Got = h.Got[:0]
	}
}

func NewCapture() (context.Context, *CaptureHandler) {
	h := &CaptureHandler{}
	ctx := event.WithExporter(context.Background(), event.NewExporter(h, ExporterOptions()))
	return ctx, h
}
