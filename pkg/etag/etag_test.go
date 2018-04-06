package etag

import (
	"bytes"
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
	if partsize > fileinfo.Size() {
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
	md5Buf := &bytes.Buffer{}
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
		if _, err := md5Buf.Write(hash); err != nil {
			return "", err
		}
		if breakFlg {
			break
		}
	}
	h := md5.New()
	if _, err := io.Copy(h, md5Buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", hex.EncodeToString(h.Sum(nil)), i), nil
}

func TestWrite(t *testing.T) {
	var vtests = []struct {
		filepath string
		partsize int64
		expected string
	}{
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
		if res != expect {
			t.Errorf("err %d:mpEtag.Sum() =%#v, want:%#v", i, res, expect)
		}
	}
}
