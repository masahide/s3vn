package s3vn

// Conf is config
type Conf struct {
	RepoName  string
	S3bucket  string
	WorkDir   string
	ConfDir   string
	MaxFiles  int
	MaxWorker int
	UserName  string
	Force     bool
	PrintLog  bool
}
