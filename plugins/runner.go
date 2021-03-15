package plugins

import "context"

type runner struct {
	Status pluginStatus
	Plugin Plugin
}

func newRunner(p Plugin) *runner {
	r := &runner{
		Plugin: p,
	}
	return r
}

func (r *runner) Start(ctx context.Context) error {
	return nil
}

func (r *runner) Stop(ctx context.Context) error {
	return nil
}
