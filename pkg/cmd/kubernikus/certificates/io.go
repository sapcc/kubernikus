package certificates

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

type ConfigPersister interface {
	WriteConfig(*v1.Kluster) error
}

type FilePersister struct {
	BaseDir string
}

func NewFilePersister(basedir string) *FilePersister {
	p := &FilePersister{}
	p.BaseDir = basedir
	return p
}

func (fp FilePersister) WriteConfig(kluster *v1.Kluster) error {
	for filename, contents := range kluster.Secret.Certificates {
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

func (fp PlainPersister) WriteConfig(kluster *v1.Kluster) error {
	for filename, contents := range kluster.Secret.Certificates {
		fmt.Println(filename)
		fmt.Println(contents)
	}

	return nil
}
