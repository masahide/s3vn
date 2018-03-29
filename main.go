package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/k0kubun/pp"
)

// FsJEvxFGrgEnbNKNodx9mAjgudA=/path/to/filename

// FileInfo file infomation
type FileInfo struct {
	Mode   os.FileMode
	Path   string
	Xxhash []byte
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

func (f *fileInfos) append(file FileInfo) {
	f.Files = append(f.Files, file)
}

func (f *fileInfos) walk(path string, info os.FileInfo, err error) error {
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
	f.append(fi)
	return nil
}

func main() {
	fs := &fileInfos{
		Files: make([]FileInfo, 0, 10000),
	}
	err := filepath.Walk(dir, fs.walk)
	if err != nil {
		fmt.Println(1, err)
	}
	pp.Print(fs)
}
