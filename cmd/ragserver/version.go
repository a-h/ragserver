package main

import (
	"context"
	"fmt"

	"github.com/a-h/ragserver"
)

type VersionCommand struct {
}

func (c VersionCommand) Run(ctx context.Context) (err error) {
	fmt.Println(ragserver.Version)
	return nil
}
