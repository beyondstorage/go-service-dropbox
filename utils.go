package dropbox

import (
	"fmt"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/auth"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"

	ps "github.com/aos-dev/go-storage/v3/pairs"
	"github.com/aos-dev/go-storage/v3/pkg/credential"
	"github.com/aos-dev/go-storage/v3/pkg/httpclient"
	"github.com/aos-dev/go-storage/v3/services"
	typ "github.com/aos-dev/go-storage/v3/types"
)

// Storage is the dropbox client.
type Storage struct {
	client files.Client

	workDir string

	pairPolicy typ.PairPolicy
}

// String implements Storager.String
func (s *Storage) String() string {
	return fmt.Sprintf(
		"Storager dropbox {WorkDir: %s}",
		s.workDir,
	)
}

// NewStorager will create Storager only.
func NewStorager(pairs ...typ.Pair) (typ.Storager, error) {
	return newStorager(pairs...)
}

// New will create a new client.
func newStorager(pairs ...typ.Pair) (store *Storage, err error) {
	defer func() {
		if err != nil {
			err = &services.InitError{Op: "new_storager", Type: Type, Err: err, Pairs: pairs}
		}
	}()

	opt, err := parsePairStorageNew(pairs)
	if err != nil {
		return
	}

	cfg := dropbox.Config{
		Client: httpclient.New(opt.HTTPClientOptions),
	}

	cred, err := credential.Parse(opt.Credential)
	if err != nil {
		return nil, err
	}

	switch cred.Protocol() {
	case credential.ProtocolAPIKey:
		cfg.Token = cred.APIKey()
	default:
		return nil, services.NewPairUnsupportedError(ps.WithCredential(opt.Credential))
	}

	store = &Storage{
		client: files.New(cfg),

		workDir: "/",
	}

	if opt.HasWorkDir {
		store.workDir = opt.WorkDir
	}
	return
}

// ref: https://www.dropbox.com/developers/documentation/http/documentation
//
// FIXME: I don't know how to handle dropbox's API error correctly, please give me some help.
func formatError(err error) error {
	fn := func(errorSummary, s string) bool {
		return strings.HasPrefix(errorSummary, s)
	}

	switch e := err.(type) {
	case files.DownloadAPIError:
		if fn(e.ErrorSummary, "not_found") {
			err = fmt.Errorf("%w: %v", services.ErrObjectNotExist, err)
		}
	case auth.AccessAPIError:
		err = fmt.Errorf("%w: %v", services.ErrPermissionDenied, err)
	}
	return err
}
func (s *Storage) getAbsPath(path string) string {
	return strings.TrimPrefix(s.workDir+"/"+path, "/")
}

func (s *Storage) formatError(op string, err error, path ...string) error {
	if err == nil {
		return nil
	}

	return &services.StorageError{
		Op:       op,
		Err:      formatError(err),
		Storager: s,
		Path:     path,
	}
}

func (s *Storage) newObject(done bool) *typ.Object {
	return typ.NewObject(s, done)
}

func (s *Storage) formatFolderObject(v *files.FolderMetadata) (o *typ.Object) {
	o = s.newObject(true)
	o.ID = v.Id
	o.Path = v.Name
	o.Mode |= typ.ModeDir

	return o
}

func (s *Storage) formatFileObject(v *files.FileMetadata) (o *typ.Object) {
	o = s.newObject(true)
	o.ID = v.Id
	o.Path = v.Name
	o.Mode |= typ.ModeRead

	o.SetContentLength(int64(v.Size))
	o.SetLastModified(v.ServerModified)
	o.SetEtag(v.ContentHash)

	return o
}
