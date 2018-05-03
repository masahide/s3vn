package etag

import (
	"crypto/md5" // nolint:gas
	"encoding/hex"
	"fmt"
	"hash"
)

type mpEtag struct {
	partsize int64
	crrSize  int64
	crrMD5   hash.Hash
	part     int
	sumMD5   hash.Hash
	lastHash []byte
}

func (m *mpEtag) Reset() {
	m.crrSize = 0
	m.crrMD5 = md5.New() // nolint:gas
	m.part = 0
	m.sumMD5 = md5.New() // nolint:gas
}

// New returns a new hash.Hash computing the MD5 checksum. The Hash also
// implements encoding.BinaryMarshaler and encoding.BinaryUnmarshaler to
// marshal and unmarshal the internal state of the hash.
func New(partsize int64) hash.Hash {
	m := new(mpEtag)
	m.partsize = partsize
	m.Reset()
	return m
}

func (m *mpEtag) Size() int { return md5.Size + 10 }

func (m *mpEtag) BlockSize() int { return md5.BlockSize }

func (m *mpEtag) Write(p []byte) (nn int, err error) {
	nn = len(p)
	// pp.Println("nn", nn)
	for {
		ws := int64(len(p))
		if m.crrSize+ws > m.partsize {
			ws = m.partsize - m.crrSize
		}
		//pp.Println("ws", ws, "m.partsize", m.partsize)
		// Writeはerrを返さない see: https://github.com/golang/go/blob/1d547e4a68f1acff6b7d1c656ea8aa665e34055f/src/crypto/md5/md5.go#L140-L161
		m.crrMD5.Write(p[:ws]) // nolint:gas,errcheck
		p = p[ws:]
		m.crrSize += ws
		if m.crrSize == m.partsize {
			m.part++
			m.lastHash = m.crrMD5.Sum(nil)
			//pp.Println("new", hash)
			m.sumMD5.Write(m.lastHash) // nolint:errcheck,gas
			m.crrMD5.Reset()
			m.crrSize = 0
		}
		if len(p) == 0 {
			return
		}
		//pp.Println("len(p)", len(p))
		//os.Exit(0)
	}
}

func (m *mpEtag) Sum(in []byte) []byte {
	if m.part == 0 && m.crrSize == 0 {
		return []byte(hex.EncodeToString(m.crrMD5.Sum(nil)))
	}
	if m.part == 1 && m.crrSize == 0 {
		return []byte(hex.EncodeToString(m.lastHash))
	}
	if m.crrSize > 0 {
		m.part++
		hash := m.crrMD5.Sum(nil)
		if m.part == 1 {
			return []byte(hex.EncodeToString(hash))
		}
		m.sumMD5.Write(hash) // nolint:errcheck,gas
	}
	str := fmt.Sprintf("%s-%d", hex.EncodeToString(m.sumMD5.Sum(nil)), m.part)
	return []byte(str)
}
