package s3vn

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"io"
	"os"
	"testing"
	"time"

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
	benchFile = "test/1mb"
	//testFile = "test/10mb"
	testFile     = "test/1mb"
	testpartSize = 5 * 1024 * 1024
)

func whashSum(prefix []byte, r io.Reader) ([]byte, uint64, error) {
	xx := xxhash.New()
	sha := sha256.New()
	w := io.MultiWriter(xx, sha)

	w.Write(prefix)
	if _, err := io.Copy(w, r); err != nil {
		return nil, 0, errors.Wrap(err, "hash sum error.")
	}
	return sha.Sum(nil), xx.Sum64(), nil
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
		prefix         []byte
		expectedEtag   string
		expectedSha256 []byte
		expectedXx     uint64
		expectedErr    *string
	}{
		{
			filepath:       testFile,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0},
			expectedEtag:   "bf38a62700dac937c420da73aa9aec27",
			expectedSha256: []byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2},
			expectedXx:     uint64(0xa183f84e7c935533),
		},
		{
			filepath:       testFile,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0, 3},
			expectedEtag:   "bf38a62700dac937c420da73aa9aec27",
			expectedSha256: []byte{0xe8, 0x79, 0x65, 0x91, 0xd3, 0x59, 0xac, 0x3b, 0x3f, 0x8b, 0x3f, 0xb4, 0xef, 0x1a, 0xef, 0x60, 0xf6, 0x63, 0x8b, 0x8f, 0x8b, 0x16, 0xe9, 0x3b, 0x82, 0xce, 0x49, 0xe0, 0xb5, 0xa5, 0x65, 0x72},
			expectedXx:     uint64(0x573929fc38a20e19),
		},
	}
	for i, vt := range vtests {
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
		if !bytes.Equal(sha, vt.expectedSha256) {
			t.Errorf("err %d:whashSum() = sha256:%#v, want:%#v", i, sha, vt.expectedSha256)
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
		sha256sum, _ := sha256Sum(vt.prefix, f)
		f.Close()
		if xx != xxsum {
			t.Errorf("err %d:whashSum() = xxSum():%#v, want:%#v", i, xx, xxsum)
		}
		if !bytes.Equal(sha, sha256sum) {
			t.Errorf("err %d:whashSum() = sha256Sum():%#v, want:%#v", i, sha, sha256sum)
		}

	}
}

func TestWhashSum(t *testing.T) {
	var vtests = []struct {
		filepath       string
		prefix         []byte
		expectedSha256 []byte
		expectedXx     uint64
		expectedErr    *string
	}{
		{
			filepath:       testFile,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0},
			expectedSha256: []byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2},
			expectedXx:     uint64(0xa183f84e7c935533),
		},
		{
			filepath:       testFile,
			prefix:         []byte{2, 3, 4, 5, 6, 7, 0, 3},
			expectedSha256: []byte{0xe8, 0x79, 0x65, 0x91, 0xd3, 0x59, 0xac, 0x3b, 0x3f, 0x8b, 0x3f, 0xb4, 0xef, 0x1a, 0xef, 0x60, 0xf6, 0x63, 0x8b, 0x8f, 0x8b, 0x16, 0xe9, 0x3b, 0x82, 0xce, 0x49, 0xe0, 0xb5, 0xa5, 0x65, 0x72},
			expectedXx:     uint64(0x573929fc38a20e19),
		},
	}
	for i, vt := range vtests {
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
		if !bytes.Equal(sha, vt.expectedSha256) {
			t.Errorf("err %d:whashSum() = sha256:%#v, want:%#v", i, sha, vt.expectedSha256)
		}
		if xx != vt.expectedXx {
			t.Errorf("err %d:whashSum() = xx:%#v, want:%#v", i, xx, vt.expectedXx)
		}
		f.Seek(0, io.SeekStart)
		xxsum, _ := xxSum(vt.prefix, f)
		f.Seek(0, io.SeekStart)
		sha256sum, _ := sha256Sum(vt.prefix, f)
		f.Close()
		if xx != xxsum {
			t.Errorf("err %d:whashSum() = xxSum():%#v, want:%#v", i, xx, xxsum)
		}
		if !bytes.Equal(sha, sha256sum) {
			t.Errorf("err %d:whashSum() = sha256Sum():%#v, want:%#v", i, sha, sha256sum)
		}

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
				{Mode: 0, Path: "a", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
			},
			new: []FileInfo{
				{Mode: 0, Path: "a", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
			},
			expected: []FileInfo{
				{Mode: 0, Path: "a", Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
			},
		},
		{
			old: []FileInfo{
				{Mode: 0, Path: "a", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
			},
			new: []FileInfo{
				{Mode: 0, Path: "a", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729272, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "b", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "c", Sha256: [32]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2}, Xxhash: 12, Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
			},
			expected: []FileInfo{
				{Mode: 0, Path: "b", Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
				{Mode: 0, Path: "c", Size: 3, Mtime: time.Unix(1522729273, 0), LinkTo: "/link/to", UID: 1, GID: 2},
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
				if fi.Path == expect.Path && !expect.Mtime.Equal(fi.Mtime) {
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
		prefix         []byte
		expectedErr    *string
		expectedXx     uint64
		expectedSha256 []byte
	}{
		{
			f:              &FileInfo{Path: testFile},
			prefix:         []byte("hoge"),
			expectedXx:     uint64(0xad1b4de420d1a738),
			expectedSha256: []byte{0x3a, 0x75, 0xca, 0x1f, 0x27, 0x43, 0x9, 0xfe, 0xed, 0x4f, 0x80, 0x6f, 0x9a, 0x87, 0x6, 0xbd, 0x3f, 0x50, 0x6c, 0xcb, 0x63, 0x29, 0x78, 0xfc, 0x71, 0x96, 0x99, 0x21, 0x61, 0xc5, 0x9, 0xc},
		},
		{
			f:           &FileInfo{Path: testFile + ".dummy"},
			prefix:      []byte("hoge"),
			expectedErr: &errMes1,
		},
	}
	for i, vt := range vtests {
		err := vt.f.getThash(vt.prefix)
		if vt.expectedErr != nil {
			if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:getWhash() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if vt.expectedSha256 != nil && !bytes.Equal(vt.expectedSha256, vt.f.Sha256[:]) {
			t.Errorf("err %d:getWhash() = %#v, want:%#v", i, vt.f.Sha256, vt.expectedSha256)
		}
		if vt.expectedXx != vt.f.Xxhash {
			t.Errorf("err %d:getWhash() = %#v, want:%#v", i, vt.f.Xxhash, vt.expectedXx)
		}
	}
}

func TestGetXxHash(t *testing.T) {
	var errMes1 = "failed getXxHash Open: open test/1mb.dummy: no such file or directory"

	var vtests = []struct {
		f           *FileInfo
		prefix      []byte
		expectedErr *string
		expectedXx  uint64
	}{
		{
			f:          &FileInfo{Path: testFile},
			prefix:     []byte("hoge"),
			expectedXx: uint64(0xad1b4de420d1a738),
		},
		{
			f:           &FileInfo{Path: testFile + ".dummy"},
			prefix:      []byte("hoge"),
			expectedErr: &errMes1,
		},
	}
	for i, vt := range vtests {
		err := vt.f.getXxHash(vt.prefix)
		if vt.expectedErr != nil {
			if err.Error() != *vt.expectedErr {
				t.Errorf("err %d:getHash() err = %#v, want:%#v", i, err, vt.expectedErr)
			}
		}
		if vt.expectedXx != vt.f.Xxhash {
			t.Errorf("err %d:getHash() = %#v, want:%#v", i, vt.f.Xxhash, vt.expectedXx)
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
				Sha256: [sha256.Size]byte{0x28, 0x18, 0x96, 0x61, 0x52, 0xf1, 0xd8, 0x45, 0xa1, 0xf, 0xa6, 0x28, 0x3d, 0xc0, 0xfd, 0xbe, 0xf8, 0x84, 0xad, 0x2, 0x1, 0x73, 0x5e, 0xcf, 0xc9, 0x98, 0x6, 0x91, 0xfe, 0x3e, 0x23, 0xf2},
				Xxhash: 111111111111111111,
				Etag:   "a9aec27937c42ad38a62700dac73a0da-224",
			},
			expected: "KBiWYVLx2EWhD6YoPcD9vviErQIBc17PyZgGkf4-I_LguTzbL5nifS_ofCQqP8Qw9cN8z7WqXA6Lpyo9--Zddw",
		},
	}
	for i, vt := range vtests {
		res := vt.f.makeKey()
		if res != vt.expected {
			t.Errorf("err %d:makeKey() = %#v, want:%#v", i, res, vt.expected)
		}
	}
}
