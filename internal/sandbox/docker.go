package sandbox

import (
	"context"
)


type dockerExecer struct {
	h *SessionHandle
}

func (d *dockerExecer) Exec(ctx context.Context, command string) (string, error) {
	return Exec(ctx, d.h, command)
}

// NewDockerExecer wraps a SessionHandle as an Execer.
func NewDockerExecer(h *SessionHandle) Execer {
	return &dockerExecer{h: h}
}
