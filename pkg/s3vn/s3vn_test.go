package s3vn

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"io"
	"log"
	"os"
	"testing"

	"github.com/cespare/xxhash"
	"github.com/masahide/s3vn/pkg/etag"
	"github.com/pkg/errors"
)

/*
func BenchmarkMakeKey(b *testing.B) {
	info, err := os.Stat("main.go")
	if err != nil {
		b.Error(err)
	}
	fi, err := mkFileInfo("./main.go", info)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeKey(fi)
	}
}
*/

const (
	benchFile  = "test/1mb"
	benchMByte = 1 //mb
	//testFile = "test/10mb"
	testFile     = "test/1mb"
	testpartSize = 5 * 1024 * 1024 //byge
)

func mkFile(path string, mb int) {
	data := bytes.Repeat([]byte{0}, 1024*1024)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	for i := 0; i <= mb; i++ {
		w.Write(data)
	}
}

func whashSum(prefix []byte, r io.Reader) ([]byte, uint64, error) {
	xx := xxhash.New()
	sha := sha512.New()
	w := io.MultiWriter(xx, sha)

	w.Write(prefix)
	if _, err := io.Copy(w, r); err != nil {
		return nil, 0, errors.Wrap(err, "hash sum error.")
	}
	return sha.Sum(nil), xx.Sum64(), nil
}

func sha512Sum(prefix []byte, r io.Reader) ([]byte, error) {
	sha := sha512.New()
	sha.Write(prefix) // errは常にnill see: https://github.com/golang/go/blob/master/src/crypto/sha512/sha512.go#L266
	if _, err := io.Copy(sha, r); err != nil {
		return nil, errors.Wrap(err, "hash sum error.")
	}
	return sha.Sum(nil), nil

}
func sha256Sum(prefix []byte, r io.Reader) ([]byte, error) {
	sha := sha256.New()
	sha.Write(prefix) // errは常にnill see: https://github.com/golang/go/blob/master/src/crypto/sha256/sha256.go#L203-L223
	if _, err := io.Copy(sha, r); err != nil {
		return nil, errors.Wrap(err, "hash sum error.")
	}
	return sha.Sum(nil), nil
}

func sha1Sum(prefix []byte, r io.Reader) ([]byte, error) {
	sha := sha1.New()
	sha.Write(prefix) // err は常にnull see: https://github.com/golang/go/blob/master/src/crypto/sha1/sha1.go#L130-L151
	if _, err := io.Copy(sha, r); err != nil {
		return nil, errors.Wrap(err, "hash sum error.")
	}
	return sha.Sum(nil), nil
}
func etagSum(partsize int64, r io.Reader) ([]byte, error) {
	et := etag.New(partsize)
	if _, err := io.Copy(et, r); err != nil {
		return nil, errors.Wrap(err, "etag sum error.")
	}
	return et.Sum(nil), nil
}

func BenchmarkThashSum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		thashSum([]byte("hogehoge 00"), f)
		f.Close()
	}
}

func BenchmarkWhashSum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		whashSum([]byte("hogehoge 00"), f)
		f.Close()
	}
}

func BenchmarkXxSum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		xxSum([]byte("hogehoge 00"), f)
		f.Close()
	}
}
func BenchmarkSha256Sum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		sha256Sum([]byte("hogehoge 00"), f)
		f.Close()
	}
}
func BenchmarkSha512Sum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		sha256Sum([]byte("hogehoge 00"), f)
		f.Close()
	}
}
func BenchmarkSha1Sum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		sha1Sum([]byte("hogehoge 00"), f)
		f.Close()
	}
}
func BenchmarkEtagSum(b *testing.B) {
	mkFile(benchFile, benchMByte)
	defer os.Remove(benchFile)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(benchFile)
		if err != nil {
			b.Error(err)
		}
		etagSum(testpartSize, f)
		f.Close()
	}
}

func TestThashSum(t *testing.T) {
	var vtests = []struct {
		filepath       string
		mb             int
		prefix         []byte
		expectedEtag   string
		expectedSha512 []byte
		expectedXx     uint64
		expectedErr    *string
	}{
		{
			filepath:       testFile,
			mb:             1,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0},
			expectedEtag:   "b2d1236c286a3c0704224fe4105eca49",
			expectedSha512: []byte{0x11, 0x11, 0x54, 0x8f, 0x1f, 0x5c, 0x0, 0x59, 0x83, 0xd3, 0xcc, 0x36, 0x8f, 0xc5, 0xbf, 0xb2, 0x21, 0x25, 0x52, 0xe7, 0xf2, 0x1c, 0x24, 0xb3, 0xc6, 0xc4, 0x25, 0x47, 0x52, 0xee, 0x9b, 0xf6, 0x7e, 0x1e, 0x71, 0x5f, 0x6d, 0xf5, 0x69, 0xb2, 0xc2, 0x72, 0x8b, 0x35, 0xbe, 0x8e, 0xa1, 0xf3, 0xa6, 0x10, 0xc9, 0x62, 0x53, 0xe1, 0xc4, 0x1e, 0xae, 0x7a, 0xe1, 0x5, 0x13, 0x56, 0x67, 0x86},
			expectedXx:     uint64(0x701214d9f354e3bb),
		},
		{
			filepath:       testFile,
			mb:             10,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0, 3},
			expectedEtag:   "041e2458cccc0bff6e10520d9e282eb1",
			expectedSha512: []byte{0x99, 0x7b, 0xb8, 0xd9, 0x81, 0x79, 0x5c, 0x97, 0xd6, 0xb8, 0xaa, 0xe, 0x47, 0x82, 0x17, 0x83, 0xad, 0xd1, 0xaa, 0xca, 0xe, 0xd6, 0x71, 0x58, 0xb0, 0x7, 0x7d, 0x60, 0xd, 0x9a, 0xe1, 0xe4, 0xcc, 0xf9, 0x1a, 0xfe, 0x5b, 0x3b, 0xc0, 0xfb, 0x2d, 0x9a, 0x84, 0x6c, 0x67, 0x22, 0xa7, 0xb3, 0x6e, 0xa0, 0x67, 0xe9, 0xe0, 0x49, 0xde, 0xf2, 0x3c, 0xc, 0x90, 0x8d, 0x83, 0x1c, 0xde, 0x64},
			expectedXx:     uint64(0x61848ce6185a1aa),
		},
	}
	for i, vt := range vtests {
		mkFile(vt.filepath, vt.mb)
		f, err := os.Open(vt.filepath)
		if err != nil {
			t.Error("can't open testfile")
		}
		etag, sha, xx, err := thashSum(vt.prefix, f)
		if vt.expectedErr != nil {
			if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:whashSum() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if !bytes.Equal(sha, vt.expectedSha512) {
			t.Errorf("err %d:whashSum() = sha512:%#v, want:%#v", i, sha, vt.expectedSha512)
		}
		if xx != vt.expectedXx {
			t.Errorf("err %d:whashSum() = xx:%#v, want:%#v", i, xx, vt.expectedXx)
		}
		if string(etag) != vt.expectedEtag {
			t.Errorf("err %d:whashSum() = etag:%s, want:%s", i, etag, vt.expectedEtag)
		}
		f.Seek(0, io.SeekStart)
		xxsum, _ := xxSum(vt.prefix, f)
		f.Seek(0, io.SeekStart)
		sha512sum, _ := sha512Sum(vt.prefix, f)
		f.Close()
		if xx != xxsum {
			t.Errorf("err %d:whashSum() = xxSum():%#v, want:%#v", i, xx, xxsum)
		}
		if !bytes.Equal(sha, sha512sum) {
			t.Errorf("err %d:whashSum() = sha512Sum():%#v, want:%#v", i, sha, sha512sum)
		}
		os.Remove(vt.filepath)
	}
}

func TestWhashSum(t *testing.T) {
	var vtests = []struct {
		filepath       string
		mb             int
		prefix         []byte
		expectedSha512 []byte
		expectedXx     uint64
		expectedErr    *string
	}{
		{
			filepath:       testFile,
			mb:             1,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0},
			expectedSha512: []byte{0x11, 0x11, 0x54, 0x8f, 0x1f, 0x5c, 0x0, 0x59, 0x83, 0xd3, 0xcc, 0x36, 0x8f, 0xc5, 0xbf, 0xb2, 0x21, 0x25, 0x52, 0xe7, 0xf2, 0x1c, 0x24, 0xb3, 0xc6, 0xc4, 0x25, 0x47, 0x52, 0xee, 0x9b, 0xf6, 0x7e, 0x1e, 0x71, 0x5f, 0x6d, 0xf5, 0x69, 0xb2, 0xc2, 0x72, 0x8b, 0x35, 0xbe, 0x8e, 0xa1, 0xf3, 0xa6, 0x10, 0xc9, 0x62, 0x53, 0xe1, 0xc4, 0x1e, 0xae, 0x7a, 0xe1, 0x5, 0x13, 0x56, 0x67, 0x86},
			expectedXx:     uint64(0x701214d9f354e3bb),
		},
		{
			filepath:       testFile,
			mb:             1,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0, 3},
			expectedSha512: []byte{0xa4, 0x4a, 0xfd, 0x63, 0xf5, 0x57, 0x40, 0x82, 0xd8, 0xd6, 0xef, 0xf1, 0x90, 0x24, 0x12, 0x2b, 0xab, 0xfe, 0xb4, 0x89, 0x27, 0x8d, 0x53, 0x50, 0xec, 0xd8, 0xcd, 0xd4, 0x7a, 0xc7, 0x83, 0xaa, 0x3b, 0xce, 0xd8, 0x62, 0x51, 0x33, 0x28, 0xdf, 0x62, 0x50, 0xcd, 0x2c, 0x8, 0x2d, 0x33, 0xfd, 0x5f, 0x1f, 0x5f, 0x20, 0x52, 0x59, 0x67, 0xde, 0xe9, 0xdb, 0x17, 0xff, 0x3e, 0x31, 0xf9, 0xc0},
			expectedXx:     uint64(0x809d98f26acc40ea),
		},
	}

	for i, vt := range vtests {
		mkFile(vt.filepath, vt.mb)
		f, err := os.Open(vt.filepath)
		if err != nil {
			t.Error("can't open testfile")
		}
		sha, xx, err := whashSum(vt.prefix, f)
		if vt.expectedErr != nil {
			if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:whashSum() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if !bytes.Equal(sha, vt.expectedSha512) {
			t.Errorf("err %d:whashSum() = sha512:%#v, want:%#v", i, sha, vt.expectedSha512)
		}
		if xx != vt.expectedXx {
			t.Errorf("err %d:whashSum() = xx:%#v, want:%#v", i, xx, vt.expectedXx)
		}
		f.Seek(0, io.SeekStart)
		xxsum, _ := xxSum(vt.prefix, f)
		f.Seek(0, io.SeekStart)
		sha512sum, _ := sha512Sum(vt.prefix, f)
		f.Close()
		if xx != xxsum {
			t.Errorf("err %d:whashSum() = xxSum():%#v, want:%#v", i, xx, xxsum)
		}
		if !bytes.Equal(sha, sha512sum) {
			t.Errorf("err %d:whashSum() = sha512Sum():%#v, want:%#v", i, sha, sha512sum)
		}
		os.Remove(vt.filepath)
	}
}

func TestMakePrefixBytes(t *testing.T) {
	var vtests = []struct {
		prefix   []byte
		size     uint64
		expected []byte
	}{
		{
			prefix:   []byte("hogefuga"),
			size:     34,
			expected: []byte{0x68, 0x6f, 0x67, 0x65, 0x66, 0x75, 0x67, 0x61, 0x20, 0x79, 0x0},
		},
		{
			prefix:   []byte("hogefuga"),
			size:     18446744073709551615,
			expected: []byte{0x68, 0x6f, 0x67, 0x65, 0x66, 0x75, 0x67, 0x61, 0x20, 0x33, 0x77, 0x35, 0x65, 0x31, 0x31, 0x32, 0x36, 0x34, 0x73, 0x67, 0x73, 0x66, 0x0},
		},
	}
	for i, vt := range vtests {
		b := makePrefixBytes(vt.prefix, vt.size)
		if !bytes.Equal(vt.expected, b) {
			t.Errorf("err %d:makePrefix() = %#v, want:%#v", i, b, vt.expected)
		}
		if !bytes.Equal(b[0:len(vt.prefix)+1], []byte(string(vt.prefix)+" ")) {
			t.Errorf("err %d:makePrefix() = prefix %#v, want:%#v", i, b[0:len(vt.prefix)+1], []byte(string(vt.prefix)+" "))

		}
		if !bytes.Equal(b[0:len(vt.prefix)+1], []byte(string(vt.prefix)+" ")) {
			t.Errorf("err %d:makePrefix() = prefix %#v, want:%#v", i, b[0:len(vt.prefix)+1], []byte(string(vt.prefix)+" "))

		}
	}
}

func TestDifference(t *testing.T) {
	var vtests = []struct {
		old      []FileInfo
		new      []FileInfo
		expected []FileInfo
	}{
		{
			old: []FileInfo{
				{Mode: 0, Path: "a", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
			},
			new: []FileInfo{
				{Mode: 0, Path: "a", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
			},
			expected: []FileInfo{
				{Mode: 0, Path: "a", Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
			},
		},
		{
			old: []FileInfo{
				{Mode: 0, Path: "a", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
			},
			new: []FileInfo{
				{Mode: 0, Path: "a", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729272, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "c", Sha512: [64]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
			},
			expected: []FileInfo{
				{Mode: 0, Path: "b", Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "c", Size: 3, Mtime: 1522729273, LinkTo: "/link/to", UID: 1, GID: 2},
			},
		},
	}
	for i, vt := range vtests {
		b := difference(vt.old, vt.new)
		if len(b) != len(vt.expected) {
			t.Errorf("err %d:len difference() = len: %#v, wantlen:%#v", i, len(b), len(vt.expected))
		}
		for _, expect := range vt.expected {
			for _, fi := range b {
				if fi.Path == expect.Path && expect.Mtime != fi.Mtime {
					t.Errorf("err %d:difference() = %#v, want:%#v", i, fi.Path, expect.Path)
				}
			}
		}
	}
}

func TestGetThash(t *testing.T) {
	var errMes1 = "failed getThash Open: open " + testFile + ".dummy: no such file or directory"

	var vtests = []struct {
		f              *FileInfo
		dummy          bool
		mb             int
		prefix         []byte
		expectedErr    *string
		expectedXx     uint64
		expectedSha512 []byte
	}{
		{
			f:              &FileInfo{Path: testFile},
			dummy:          false,
			mb:             1,
			prefix:         []byte("hoge"),
			expectedXx:     uint64(0x45b0dad7d7c5bdb5),
			expectedSha512: []byte{0xb4, 0xe6, 0x6, 0x26, 0xb9, 0x95, 0xbf, 0x4c, 0xf6, 0xe7, 0x9, 0xd2, 0x94, 0x30, 0x9d, 0x80, 0xfa, 0xe, 0xbc, 0x97, 0x68, 0x90, 0xd8, 0xee, 0xa1, 0xb6, 0x7a, 0xef, 0x7a, 0x88, 0x8f, 0xe9, 0x8f, 0xdc, 0x8b, 0xc0, 0x21, 0x74, 0xf0, 0xf9, 0xaf, 0xd4, 0x3e, 0xf5, 0x91, 0x21, 0x53, 0xb1, 0x76, 0x43, 0xfb, 0xb5, 0xa7, 0xc5, 0x95, 0x46, 0x78, 0x66, 0x2, 0xdb, 0xaf, 0x68, 0x2a, 0x88},
		},
		{
			f:           &FileInfo{Path: testFile + ".dummy"},
			dummy:       true,
			mb:          1,
			prefix:      []byte("hoge"),
			expectedErr: &errMes1,
		},
	}
	for i, vt := range vtests {
		if !vt.dummy {
			mkFile(vt.f.Path, vt.mb)
		}
		err := vt.f.getThash(vt.prefix)
		if vt.expectedErr != nil {
			if err == nil {
				t.Errorf("err %d:getWhash() err = %#v , want:%#v", i, err, vt.expectedErr)
			} else if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:getWhash() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if vt.expectedSha512 != nil && !bytes.Equal(vt.expectedSha512, vt.f.Sha512[:]) {
			t.Errorf("err %d:getWhash() = %#v, want:%#v", i, vt.f.Sha512, vt.expectedSha512)
		}
		if vt.expectedXx != vt.f.Xxhash {
			t.Errorf("err %d:getWhash() = %#v, want:%#v", i, vt.f.Xxhash, vt.expectedXx)
		}
		if !vt.dummy {
			os.Remove(vt.f.Path)
		}
	}
}

func TestGetXxHash(t *testing.T) {
	var errMes1 = "failed getXxHash Open: open test/1mb.dummy: no such file or directory"

	var vtests = []struct {
		f           *FileInfo
		dummy       bool
		mb          int
		prefix      []byte
		expectedErr *string
		expectedXx  uint64
	}{
		{
			f:          &FileInfo{Path: testFile},
			mb:         1,
			prefix:     []byte("hoge"),
			expectedXx: uint64(0x45b0dad7d7c5bdb5),
		},
		{
			f:           &FileInfo{Path: testFile + ".dummy"},
			dummy:       true,
			mb:          1,
			prefix:      []byte("hoge"),
			expectedErr: &errMes1,
		},
	}
	for i, vt := range vtests {
		if !vt.dummy {
			mkFile(vt.f.Path, vt.mb)
		}
		err := vt.f.getXxHash(vt.prefix)
		if vt.expectedErr != nil {
			if err == nil {
				t.Errorf("err %d:getWhash() err = %#v , want:%#v", i, err, vt.expectedErr)
			} else if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:getHash() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if vt.expectedXx != vt.f.Xxhash {
			t.Errorf("err %d:getHash() = %#v, want:%#v", i, vt.f.Xxhash, vt.expectedXx)
		}
		if !vt.dummy {
			os.Remove(vt.f.Path)
		}
	}
}

func TestBase64URLSafe(t *testing.T) {
	var vtests = []struct {
		b        []byte
		expected string
	}{
		{b: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			expected: "AQIDBAUGBwgJAAECAwQFBgcICQABAgMEBQYHCAkAAQIDBAUGBwgJAA",
		},
	}
	for i, vt := range vtests {
		res := base64URLSafe(vt.b)
		if res != vt.expected {
			t.Errorf("err %d:base64URLSafe() = %#v, want:%#v", i, res, vt.expected)
		}
	}
}

func TestMakeKey(t *testing.T) {
	var vtests = []struct {
		f        *FileInfo
		expected string
	}{
		{
			f: &FileInfo{
				Sha512: [sha512.Size]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2, 0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2},
				Xxhash: 111111111111111111,
				Etag:   "a9aec27937c42ad38a62700dac73a0da-224",
			},
			expected: "KBiWYVLx2EWhD6YoPcD9vviErQIBc17PyZgGkf4-I_IoGJZhUvHYRaEPpig9wP2--IStAgFzXs_JmAaR_j4j8sdxYIT3vooBqa7CeTfEKtOKYnANrHOg2g",
		},
	}
	for i, vt := range vtests {
		res := vt.f.makeKey()
		if res != vt.expected {
			t.Errorf("err %d:makeKey() = %#v, want:%#v", i, res, vt.expected)
		}
	}
}

func TestMd5StringToBytes(t *testing.T) {
	var vtests = []struct {
		s        string
		expected []byte
		err      bool
	}{
		{
			s:        "b0804ec967f48520697662a204f5fe72",
			expected: []byte{0xb0, 0x80, 0x4e, 0xc9, 0x67, 0xf4, 0x85, 0x20, 0x69, 0x76, 0x62, 0xa2, 0x4, 0xf5, 0xfe, 0x72},
			err:      false,
		},
		{
			s:        "23804ec967f48520697662a204f5fe11-222",
			expected: []byte{0x23, 0x80, 0x4e, 0xc9, 0x67, 0xf4, 0x85, 0x20, 0x69, 0x76, 0x62, 0xa2, 0x4, 0xf5, 0xfe, 0x11},
			err:      false,
		},
		{
			s:   "ab",
			err: true,
		},
		{
			s:   "23804e-c967f48520697662a204f5fe11-222",
			err: true,
		},
	}
	for i, vt := range vtests {
		res, err := md5StringToBytes(vt.s)
		if vt.err && err == nil {
			t.Errorf("err %d:md5StringToBytes() = %#v, want error", i, res)
		} else if !bytes.Equal(res, vt.expected) {
			t.Errorf("err %d:md5StringToBytes() = %#v, want:%#v", i, res, vt.expected)
		}
	}

}

func TestMakeFileInfos(t *testing.T) {
	sn := &S3vn{
		Files: FileInfos{},
	}
	sn.makeFileInfos("./test")
	//pp.Print(sn)
}

/*
&s3vn.S3vn{
	Files: []s3vn.FileInfo{
		s3vn.FileInfo{
			Mode:   0x000001b4,
			Path:   "test/a",
			Sha256: [32]uint8{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			Xxhash: 0x0000000000000000,
			Etag:   "",
			S3Key:  "",
			Size:   136373,
			Mtime:  2018-04-03 09:50:26 Local,
			LinkTo: "",
			UID:    0x000006a2,
			GID:    0x000006a2,
		},
		s3vn.FileInfo{
			Mode:   0x080001ff,
			Path:   "test/bb",
			Sha256: [32]uint8{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			Xxhash: 0x0000000000000000,
			Etag:   "",
			S3Key:  "",
			Size:   2,
			Mtime:  2018-04-22 18:41:14 Local,
			LinkTo: "aa",
			UID:    0x000006a2,
			GID:    0x000006a2,
		},
	},
	s3m:  (*s3manager.Uploader)(nil),
	Conf: s3vn.Conf{
		RepoName:  "",
		S3bucket:  "",
		WorkDir:   "",
		ConfDir:   "",
		MaxFiles:  0,
		MaxWorker: 0,
		UserName:  "",
		Force:     false,
		PrintLog:  false,
	},
}-
*/
