package main

import (
	"os"
	"path"

	"github.com/pkg/sftp"
)

func putDir(src, dst string, sc *sftp.Client) error {
	var err error
	var fds []os.FileInfo

	if err = sc.Mkdir(dst); err != nil {
		return err
	}

	if fds, err = sc.ReadDir(src); err != nil {
		return err
	}

	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = putDir(srcfp, dstfp, sc); err != nil {
				return err
			}
			if err = putFile(srcfp, dstfp, sc); err != nil {
				return err
			}
		}
	}
	return nil
}

func putFile(src, dst string, sc *sftp.Client) error {}

// place in current folder
func getDir(src string, sc *sftp.Client) error {}

func getFile(src, dst string, sc *sftp.Client) error {}
