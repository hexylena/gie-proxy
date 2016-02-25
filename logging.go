package main

import (
	"github.com/op/go-logging"
	"os"
)

var log = logging.MustGetLogger("main")

func setupLogging() {
	format := logging.MustStringFormatter(
		"%{color}%{time:15:04:05.000} %{shortfunc} > %{level:.4s} %{id:03x}%{color:reset} %{message}",
	)
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.DEBUG, "")
	logging.SetFormatter(format)
	log.SetBackend(backend1Leveled)
}
