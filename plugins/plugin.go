package plugins

import "context"

type Plugin interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
}
