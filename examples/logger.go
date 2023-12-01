package examples

import (
	"github.com/go-logr/logr"
	"github.com/rdimitrov/go-tuf-metadata/metadata"
)

// WLogger is a wrapper around logr.Logger so it can be used as
// metadata.Logger.
type WLogger struct {
	logr.Logger
}

// V returns an updated logger with the provided loglevel
func (w WLogger) V(level int) metadata.Logger {
	var cp logr.Logger
	cp = w.Logger
	cp = cp.V(level)

	return WLogger{cp}
}
