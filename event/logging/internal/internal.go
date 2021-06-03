package internal

import (
	"context"
	"time"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/keys"
)

var (
	// TODO: these should be in event/keys.
	LevelKey = keys.Int("level")
	NameKey  = keys.String("name")
	ErrorKey = keys.Value("error")
)

type TestHandler struct {
	Got event.Event
}

func (h *TestHandler) Log(_ context.Context, ev *event.Event) {
	h.Got = *ev
	h.Got.Labels = make([]event.Label, len(ev.Labels))
	copy(h.Got.Labels, ev.Labels)
}

func (h *TestHandler) Annotate(_ context.Context, _ *event.Event)                {}
func (h *TestHandler) Metric(_ context.Context, _ *event.Event)                  {}
func (h *TestHandler) Start(ctx context.Context, _ *event.Event) context.Context { return ctx }
func (h *TestHandler) End(_ context.Context, _ *event.Event)                     {}

var TestAt = time.Now()

func NewTestExporter() (*event.Exporter, *TestHandler) {
	te := &TestHandler{}
	opts := event.ExporterOptions{Now: func() time.Time { return TestAt }}
	return opts.NewExporter(te), te
}
