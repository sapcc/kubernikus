package certificates

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type ConfigPersister interface {
	WriteConfig(map[string]string) error
}

type FilePersister struct {
	BaseDir string
}

func NewFilePersister(basedir string) *FilePersister {
	p := &FilePersister{}
	p.BaseDir = basedir
	return p
}

func (fp FilePersister) WriteConfig(certificates map[string]string) error {
	for filename, contents := range certificates {
		if err := write(path.Join(fp.BaseDir, filename), []byte(contents)); err != nil {
			return err
		}
	}

	return nil
}

func write(certPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(certPath), os.FileMode(0755)); err != nil {
		return err
	}
	if err := ioutil.WriteFile(certPath, data, os.FileMode(0644)); err != nil {
		return err
	}
	return nil
}

type PlainPersister struct{}

func NewPlainPersister() *PlainPersister {
	return &PlainPersister{}
}

func (fp PlainPersister) WriteConfig(certificates map[string]string) error {
	for filename, contents := range certificates {
		fmt.Println(filename)
		fmt.Println(contents)
	}

	return nil
}
