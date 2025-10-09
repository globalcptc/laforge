//go:build debug

package main

import (
	"github.com/kardianos/service"
)

func logf(l service.Logger, t string, msg string, args ...interface{}) {
	switch t {
	case LogError:
		l.Errorf(msg, args)
	case LogWarning:
		l.Warningf(msg, args)
	case LogInfo:
		l.Infof(msg, args)
	default:
		l.Errorf("Unknown log type (%s) provided: '%s' %v", t, msg, args)
	}
}

func log(l service.Logger, t string, msg string) {
	switch t {
	case LogError:
		l.Error(msg)
	case LogWarning:
		l.Warning(msg)
	case LogInfo:
		l.Info(msg)
	default:
		l.Errorf("Unknown log type (%s) provided: '%s'", t, msg)
	}
}
