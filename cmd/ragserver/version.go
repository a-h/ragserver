package main

import (
	"context"
	"fmt"
)

var version string

type VersionCommand struct {
}

func (c VersionCommand) Run(ctx context.Context) (err error) {
	fmt.Println(version)
	return nil
}
