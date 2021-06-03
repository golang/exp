package internal

import (
	"context"

	"golang.org/x/exp/event"
	"golang.org/x/exp/event/adapter/eventtest"
	"golang.org/x/exp/event/keys"
)

var (
	// TODO: these should be in event/keys.
	LevelKey = keys.Int("level")
	NameKey  = keys.String("name")
	ErrorKey = keys.Value("error")
)

type TestHandler struct {
	event.NopHandler
	Got event.Event
}

func (h *TestHandler) Log(_ context.Context, ev *event.Event) {
	h.Got = *ev
	h.Got.Labels = make([]event.Label, len(ev.Labels))
	copy(h.Got.Labels, ev.Labels)
}

func NewTestExporter() (*event.Exporter, *TestHandler) {
	te := &TestHandler{}
	return event.NewExporter(te, eventtest.ExporterOptions()), te
}
