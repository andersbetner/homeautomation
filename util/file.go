package util

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
)

func CopyDir(dst string, src string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, f := range files {
		CopyFile(path.Join(dst, f.Name()), path.Join(src, f.Name()))
	}
	return nil
}
func CopyFile(dst string, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	num, err := io.Copy(out, in)
	if num < 1 {
		return errors.New("0 bytes copied")
	}
	if err != nil {
		return err
	}
	err = out.Close()
	if err != nil {
		return err
	}
	return nil
}
