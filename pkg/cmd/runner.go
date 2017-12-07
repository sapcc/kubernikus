package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/databus23/guttle/group"
)

func Runner() *group.Group {
	var g group.Group

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
