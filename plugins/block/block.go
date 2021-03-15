package main

import (
	"context"
	"fmt"
)

type block struct{}

func (b block) Name() string {
	return "block"
}
func (b block) Start(ctx context.Context) error {
	fmt.Println("block plugin start")
	return nil
}

func (b block) Stop(ctx context.Context) error {
	fmt.Println("block plugin stop")
	return nil
}

// exported
var Plugin block
