package internal

import (
	"golang.org/x/exp/event/keys"
)

var (
	// TODO: these should be in event/keys.
	LevelKey = keys.Int("level")
	NameKey  = keys.String("name")
	ErrorKey = keys.Value("error")
)
