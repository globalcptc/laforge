// +build !debug

package main

import (
	"github.com/kardianos/service"
)

func logf(l service.Logger, t string, msg string, args ...interface{}) { }

func log(l service.Logger, t string, msg string) { }
