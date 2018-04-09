package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
)

func Runner() *run.Group {
	var g run.Group

	sigs := make(chan os.Signal, 1)

	g.Add(func() error {
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		<-sigs
		return nil
	}, func(error) {
		close(sigs)
	})
	return &g

}
