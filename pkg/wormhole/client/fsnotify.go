package client

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

type fsNotify struct {
	watcher  *fsnotify.Watcher
	filename string
}

func NewFSNotify(filename string) (*fsNotify, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(filename)
	if err != nil {
		return nil, err
	}

	fsn := fsNotify{
		watcher:  watcher,
		filename: filename,
	}

	return &fsn, nil
}

func (fsn *fsNotify) Start(stopCh <-chan struct{}) error {
	defer fsn.watcher.Close()
	for {
		select {
		case event, ok := <-fsn.watcher.Events:
			if ok && event.Op&fsnotify.Write == fsnotify.Write {
				return fmt.Errorf("File change detected: %s", fsn.filename)
			}
		case <-stopCh:
			return nil
		}
	}
}
