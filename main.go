package main

import (
	"crypto/sha1"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

const (
	HashSize = sha1.Size
)

// FsJEvxFGrgEnbNKNodx9mAjgudA=/path/to/filename

// FileInfo file infomation
type FileInfo struct {
	Mode   os.FileMode
	Path   string
	Hash   [HashSize]byte
	Size   int64
	Mtime  time.Time
	LinkTo string
	UID    uint32
	GID    uint32
}

var dir = "./"

type fileInfos struct {
	Files []FileInfo
}

func makeHash(path string) ([]byte, error) {
	res := []byte{}
	f, err := os.Open(path)
	if err != nil {
		return res, err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return res, err
	}
	return h.Sum(nil), nil
}

func (fs *fileInfos) append(file FileInfo) {
	fs.Files = append(fs.Files, file)
}

func (fs *fileInfos) walk(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		// 特定のディレクトリ以下を無視する場合は
		// return filepath.SkipDir
		return nil
	}
	fi := FileInfo{
		Mode:  info.Mode(),
		Size:  info.Size(),
		Mtime: info.ModTime(),
		Path:  path,
		UID:   info.Sys().(*syscall.Stat_t).Uid,
		GID:   info.Sys().(*syscall.Stat_t).Gid,
	}
	if fi.Mode&os.ModeSymlink != 0 {
		fi.LinkTo, err = os.Readlink(path)
		if err != nil {
			return err
		}
	}
	fs.append(fi)
	return nil
}

func main() {
	makeFileInfos(os.Args[1])
}

func makeFileInfos(dir string) ([]FileInfo, error) {
	fs := &fileInfos{
		Files: make([]FileInfo, 0, 10000),
	}
	err := filepath.Walk(dir, fs.walk)
	if err != nil {
		return nil, errors.Wrap(err, "Failed makeFileInfos")
	}
	return fs.Files, nil
}
