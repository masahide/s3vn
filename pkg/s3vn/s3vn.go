package s3vn

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cespare/xxhash"
	"github.com/masahide/s3vn/pkg/etag"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	//	limit    = 24                // 同時実行数の上限
	//	weight   = 1                 // 1処理あたりの実行コスト
	partSize = 100 * 1024 * 1024 // 100MB
)

var (
	maxWorkers = runtime.GOMAXPROCS(0)
)

// FsJEvxFGrgEnbNKNodx9mAjgudA=/path/to/filename

// FileInfo file infomation
type FileInfo struct {
	Mode   os.FileMode
	Path   string
	Sha256 [sha256.Size]byte
	Xxhash uint64
	Etag   string
	S3Key  string
	Size   int64
	Mtime  time.Time
	LinkTo string
	UID    uint32
	GID    uint32
	//Gzip   bool
	//Kms    bool
}

func (f *FileInfo) makeKey() string {
	// base64(join(Sha256, sha256.sum(XxHash, Etag)))
	b := make([]byte, 8+len(f.Etag))
	binary.LittleEndian.PutUint64(b, f.Xxhash)
	copy(b[8:], []byte(f.Etag))
	h := sha256.New()
	h.Write(b) // nolint:errcheck
	// Writeはerrを返さない see: https://github.com/golang/go/blob/1d547e4a68f1acff6b7d1c656ea8aa665e34055f/src/crypto/sha256/sha256.go#L195-L216
	res := make([]byte, len(f.Sha256)*2)
	copy(res, f.Sha256[:])
	copy(res[len(f.Sha256):], h.Sum(nil)[:])
	return base64URLSafe(res)
}

func (f *FileInfo) getThash(prefix []byte) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return errors.Wrap(err, "failed getThash Open")
	}
	defer file.Close() // nolint:errcheck
	etag, sha, xx, err := thashSum(makePrefixBytes(prefix, uint64(f.Size)), file)
	if err != nil {
		return errors.Wrap(err, "failed hashSum")
	}
	f.Xxhash = xx
	f.Etag = string(etag)
	copy(f.Sha256[:], sha)
	return nil

}

func (f *FileInfo) getXxHash(prefix []byte) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return errors.Wrap(err, "failed getXxHash Open")
	}
	defer file.Close() // nolint:errcheck
	xx, err := xxSum(makePrefixBytes(prefix, uint64(f.Size)), file)
	if err != nil {
		return errors.Wrap(err, "failed xxSum")
	}
	f.Xxhash = xx
	return nil

}

// FileInfos slice of FileInfo
type FileInfos []FileInfo

// 以下インタフェースを満たす

func (f FileInfos) Len() int {
	return len(f)
}

func (f FileInfos) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FileInfos) Less(i, j int) bool {
	return f[i].Path < f[j].Path
}

// S3vn is filelist of s3vn dir infomation
type S3vn struct {
	Files []FileInfo
	s3m   *s3manager.Uploader
	Conf
}

func (sn *S3vn) append(file FileInfo) {
	sn.Files = append(sn.Files, file)
}

func (sn *S3vn) makeFileInfos(dir string) error {
	sn.Files = make(FileInfos, 0, 10000)
	if err := filepath.Walk(dir, sn.walk); err != nil {
		return errors.Wrap(err, "Failed makeFileInfos.")
	}
	return nil
}

func (sn *S3vn) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		// 特定のディレクトリ以下を無視する場合は return filepath.SkipDir
		return nil
	}
	fi, err := mkFileInfo(path, info)
	if err != nil {
		return err
	}
	sn.append(fi)
	return nil
}

func (sn *S3vn) reHashCommit(ctx context.Context, files FileInfos) error {
	if sn.MaxWorker == 0 {
		sn.MaxWorker = maxWorkers
	}
	prefix := []byte(sn.RepoName)
	sem := semaphore.NewWeighted(int64(sn.MaxWorker))
	g, ctx := errgroup.WithContext(ctx)
	for i := range files {
		fi := &files[i]
		if err := sem.Acquire(ctx, 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}
		g.Go(func() error {
			defer sem.Release(1)
			if err := fi.getThash(prefix); err != nil {
				return err
			}
			return sn.upload(ctx, fi)
			//return nil
		})
	}
	return g.Wait()
}

func (sn *S3vn) upload(ctx context.Context, fi *FileInfo) error {
	f, err := os.Open(fi.Path)
	if err != nil {
		return errors.Wrap(err, "failed upload os.Open")
	}
	defer f.Close() // nolint:errcheck
	fi.S3Key = fi.makeKey()
	if fi.Size > partSize {
		return sn.multipartUpload(ctx, f, fi)
	}
	input := &s3.PutObjectInput{
		Body:   f,
		Bucket: &sn.S3bucket,
		Key:    &fi.S3Key,
	}
	res, err := sn.s3m.S3.PutObject(input)
	if err != nil {
		return errors.Wrapf(err, "PutObject:%s, path:%s", fi.S3Key, fi.Path)
	}
	//pp.Println(sn.Conf, *res.ETag, fi.Etag)
	if *res.ETag != "\""+fi.Etag+"\"" {
		return fmt.Errorf("Failed PutObject:%s, Etag is different. path%s, res.etag:%s, fi.etag:%s", fi.S3Key, fi.Path, *res.ETag, fi.Etag)
	}
	if sn.Conf.PrintLog {
		fmt.Printf("upload: %s -> %s/%s\n", fi.Path, sn.Conf.S3bucket, fi.S3Key)
	}
	return nil
}

func (sn *S3vn) multipartUpload(ctx context.Context, f io.Reader, fi *FileInfo) error {
	upParams := &s3manager.UploadInput{
		Bucket: &sn.S3bucket,
		Key:    &fi.S3Key,
		Body:   f,
	}
	_, err := sn.s3m.UploadWithContext(ctx, upParams)
	if err != nil {
		return errors.Wrapf(err, "s3manager.Upload:%s, path:%s", fi.S3Key, fi.Path)
	}
	input := &s3.HeadObjectInput{Bucket: &sn.S3bucket, Key: &fi.S3Key}
	result, err := sn.s3m.S3.HeadObject(input)
	if err != nil {
		return errors.Wrapf(err, "Failed HeadObject:%s, path:%s", fi.S3Key, fi.Path)
	}
	log.Printf("%s == %s", *result.ETag, fi.Etag)
	if *result.ETag != "\""+fi.Etag+"\"" {
		return fmt.Errorf("Failed PutObject:%s, Etag is different. path%s, etag:%s", fi.S3Key, fi.Path, *result.ETag)
	}
	if sn.Conf.PrintLog {
		fmt.Printf("s3manager.upload: %s -> %s/%s\n", fi.Path, sn.Conf.S3bucket, fi.S3Key)
	}
	return nil
}

// New S3vn struct
func New(sess client.ConfigProvider, conf Conf) *S3vn {
	sn := &S3vn{
		Files: make(FileInfos, 0, conf.MaxFiles),
		s3m:   s3manager.NewUploader(sess, func(u *s3manager.Uploader) { u.PartSize = partSize }),
		Conf:  conf,
	}
	return sn
}

// Commit is makeFIleinfos and reHashCommit
func (sn *S3vn) Commit(ctx context.Context, path string) {
	// stage1
	if err := sn.makeFileInfos(path); err != nil {
		log.Fatal(err)
	}
	sum := int64(0)
	count := 0
	for _, fi := range sn.Files {
		sum += fi.Size
		count++
	}
	log.Printf("count:%d, sum size:%d", count, sum)

	// stage2
	// TODO: 前回からの変更差分取得
	old := FileInfos{}
	diff := difference(old, sn.Files)

	// stage3
	// 差分アップロード
	err := sn.reHashCommit(ctx, diff)
	log.Printf("reHashCommit: %+v", err)

	// stage4
	// TODO:リストアップロード
	//pp.Println(diff) // nolint:errcheck
}

func difference(old, new FileInfos) FileInfos {
	if len(old) == 0 {
		return new
	}
	newMap := map[FileInfo]int{}
	for i, fi := range new {
		fi.Sha256 = [sha256.Size]byte{}
		fi.Xxhash = 0
		newMap[fi] = i
	}
	for _, fi := range old {
		fi.Sha256 = [sha256.Size]byte{}
		fi.Xxhash = 0
		_, ok := newMap[fi]
		if ok {
			delete(newMap, fi)
		}
	}
	res := make(FileInfos, len(newMap))
	i := 0
	for k := range newMap {
		res[i] = k
		i++
	}
	return res
}

func thashSum(prefix []byte, r io.Reader) ([]byte, []byte, uint64, error) {
	et := etag.New(partSize)
	xx := xxhash.New()
	sha := sha256.New()
	w := io.MultiWriter(xx, sha)
	t := io.MultiWriter(w, et)

	// xx,sha 共にエラーを返さない
	w.Write(prefix) // nolint:errcheck
	if _, err := io.Copy(t, r); err != nil {
		return nil, nil, 0, errors.Wrap(err, "hash sum error.")
	}
	return et.Sum(nil), sha.Sum(nil), xx.Sum64(), nil
}

func xxSum(prefix []byte, r io.Reader) (uint64, error) {
	xx := xxhash.New()
	xx.Write(prefix) // nolint:errcheck
	// errは常にnil see: https://github.com/cespare/xxhash/blob/master/xxhash.go#L62-L94
	if _, err := io.Copy(xx, r); err != nil {
		return 0, errors.Wrap(err, "hash sum error.")
	}
	return xx.Sum64(), nil
}

func makePrefixBytes(prefix []byte, size uint64) []byte {
	s := strconv.FormatUint(size, 36) // 36進数
	b := make([]byte, len(prefix)+len(s)+2)
	b[len(prefix)] = byte(' ')
	b[len(b)-1] = 0
	copy(b[:len(prefix)], prefix)
	copy(b[len(prefix)+1:], s)
	return b
}

func base64URLSafe(r []byte) string {
	s := base64.StdEncoding.EncodeToString(r)
	return strings.TrimRight(strings.NewReplacer("+", "-", "/", "_").Replace(s), "=")
}

func mkFileInfo(path string, info os.FileInfo) (FileInfo, error) {
	fi := FileInfo{
		Mode:  info.Mode(),
		Size:  info.Size(),
		Mtime: info.ModTime(),
		Path:  path,
		UID:   info.Sys().(*syscall.Stat_t).Uid,
		GID:   info.Sys().(*syscall.Stat_t).Gid,
	}
	if fi.Mode&os.ModeSymlink != 0 {
		var err error
		fi.LinkTo, err = os.Readlink(path)
		if err != nil {
			return fi, err
		}
	}
	return fi, nil
}
