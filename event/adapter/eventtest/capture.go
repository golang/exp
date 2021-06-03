package eventtest

import (
	"context"

	"golang.org/x/exp/event"
)

type CaptureHandler struct {
	Got []event.Event
}

func (h *CaptureHandler) Log(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *CaptureHandler) Metric(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *CaptureHandler) Annotate(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *CaptureHandler) Start(ctx context.Context, ev *event.Event) context.Context {
	h.event(ctx, ev)
	return ctx
}

func (h *CaptureHandler) End(ctx context.Context, ev *event.Event) {
	h.event(ctx, ev)
}

func (h *CaptureHandler) Reset() {
	if len(h.Got) > 0 {
		h.Got = h.Got[:0]
	}
}

func (h *CaptureHandler) event(ctx context.Context, ev *event.Event) {
	h.Got = append(h.Got, *ev)
	got := &h.Got[len(h.Got)-1]
	got.Labels = make([]event.Label, len(ev.Labels))
	copy(got.Labels, ev.Labels)
}

func NewCapture() (context.Context, *CaptureHandler) {
	h := &CaptureHandler{}
	ctx := event.WithExporter(context.Background(), event.NewExporter(h, ExporterOptions()))
	return ctx, h
}
