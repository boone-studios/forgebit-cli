package forgebit

import "context"

type Status struct {
	Source  string
	Details string
}

type Backend interface {
	Ping(ctx context.Context) (Status, error)
}
