package etag

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
)

func multipartEtag(filepath string, partsize int64) (string, error) {
	fileinfo, err := os.Stat(filepath)
	if err != nil {
		log.Fatal(err)
	}
	if partsize >= fileinfo.Size() {
		f, _ := os.Open(filepath)
		h := md5.New()
		io.Copy(h, f)
		return hex.EncodeToString(h.Sum(nil)), nil
	}
	f, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sum := md5.New()
	var i int
	var breakFlg bool
	for {
		h := md5.New()
		w, err := io.CopyN(h, f, partsize)
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			if w == 0 {
				break
			}
			breakFlg = true
		}
		i++
		hash := h.Sum(nil)
		//pp.Println("org", hash)
		sum.Write(hash)
		if breakFlg {
			break
		}
	}
	return fmt.Sprintf("%s-%d", hex.EncodeToString(sum.Sum(nil)), i), nil
}

func TestWrite(t *testing.T) {
	var vtests = []struct {
		filepath string
		partsize int64
		expected string
	}{
		{filepath: "test/0byte", partsize: int64(1024)},
		{filepath: "test/1mb", partsize: int64(1024)},
		{filepath: "test/1mb", partsize: int64(1024 - 1)},
		{filepath: "test/1mb", partsize: int64(1024 + 1)},
		{filepath: "test/1mb", partsize: int64(1024 * 1024)},
		{filepath: "test/1mb", partsize: int64(1024*1024 - 1)},
		{filepath: "test/1mb", partsize: int64(1024*1024 + 1)},
	}
	for i, vt := range vtests {
		expect, err := multipartEtag(vt.filepath, vt.partsize)
		if err != nil {
			t.Error(err)
		}
		h := New(vt.partsize)
		f, err := os.Open(vt.filepath)
		if err != nil {
			t.Error(err)
		}
		_, err = io.Copy(h, f)
		if err != nil {
			t.Error(err)
		}
		res := string(h.Sum(nil))
		//log.Printf("%d:mpEtag.Sum() =%#v, want:%#v", i, res, expect)
		if res != expect {
			t.Errorf("err %d:mpEtag.Sum() =%#v, want:%#v", i, res, expect)
		}
	}
}
