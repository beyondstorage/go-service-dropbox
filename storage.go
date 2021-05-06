package dropbox

import (
	"context"
	"fmt"
	"io"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"

	"github.com/aos-dev/go-storage/v3/pkg/iowrap"
	. "github.com/aos-dev/go-storage/v3/types"
)

func (s *Storage) commitAppend(ctx context.Context, o *Object, opt pairStorageCommitAppend) (err error) {
	if !o.Mode.IsAppend() {
		err = fmt.Errorf("object not appendable")
		return
	}

	rp := o.GetID()

	offset, ok := o.GetAppendOffset()
	if !ok {
		err = fmt.Errorf("append offset is not set")
		return
	}

	sessionId := GetObjectMetadata(o).UploadSessionID

	cursor := &files.UploadSessionCursor{
		SessionId: sessionId,
		Offset:    uint64(offset),
	}

	input := &files.CommitInfo{
		Path: rp,
		Mode: &files.WriteMode{
			Tagged: dropbox.Tagged{
				Tag: files.WriteModeAdd,
			},
		},
	}

	finishArg := &files.UploadSessionFinishArg{
		Cursor: cursor,
		Commit: input,
	}

	fileMetadata, err := s.client.UploadSessionFinish(finishArg, nil)

	if err == nil {
		o.Mode &= ^ModeAppend
		if fileMetadata != nil && fileMetadata.IsDownloadable {
			o.Mode |= ModeRead
		}
	}

	return err
}

func (s *Storage) create(path string, opt pairStorageCreate) (o *Object) {
	o = s.newObject(false)
	o.Mode = ModeRead
	o.ID = s.getAbsPath(path)
	o.Path = path
	return o
}

func (s *Storage) createAppend(ctx context.Context, path string, opt pairStorageCreateAppend) (o *Object, err error) {
	startArg := &files.UploadSessionStartArg{
		Close: false,
	}

	res, err := s.client.UploadSessionStart(startArg, nil)
	if err != nil {
		return
	}

	if res == nil {
		err = fmt.Errorf("upload session start response is nil")
		return
	}

	sm := ObjectMetadata{
		UploadSessionID: res.SessionId,
	}

	o = s.newObject(true)
	o.Mode = ModeAppend
	o.ID = s.getAbsPath(path)
	o.Path = path
	o.SetAppendOffset(0)
	o.SetServiceMetadata(sm)
	return o, nil
}

func (s *Storage) delete(ctx context.Context, path string, opt pairStorageDelete) (err error) {
	rp := s.getAbsPath(path)

	input := &files.DeleteArg{
		Path: rp,
	}

	_, err = s.client.DeleteV2(input)
	if err != nil {
		if deleteErr, ok := err.(files.DeleteV2APIError); ok {
			if deleteErr.EndpointError.PathLookup.Tag == files.LookupErrorNotFound {
				return nil
			}
		}
		return err
	}

	return nil
}

func (s *Storage) list(ctx context.Context, path string, opt pairStorageList) (oi *ObjectIterator, err error) {
	input := &objectPageStatus{
		limit: 200,
		path:  s.getAbsPath(path),
	}

	if opt.ListMode.IsPrefix() {
		input.recursive = true
	}

	return NewObjectIterator(ctx, s.nextObjectPage, input), nil
}

func (s *Storage) metadata(ctx context.Context, opt pairStorageMetadata) (meta *StorageMeta, err error) {
	meta = NewStorageMeta()
	meta.WorkDir = s.workDir
	meta.Name = ""

	return
}

func (s *Storage) nextObjectPage(ctx context.Context, page *ObjectPage) error {
	input := page.Status.(*objectPageStatus)

	var err error
	var output *files.ListFolderResult

	if input.cursor == "" {
		output, err = s.client.ListFolder(&files.ListFolderArg{
			Path: input.path,
		})
	} else {
		output, err = s.client.ListFolderContinue(&files.ListFolderContinueArg{
			Cursor: input.cursor,
		})
	}
	if err != nil {
		return err
	}

	for _, v := range output.Entries {
		var o *Object
		switch meta := v.(type) {
		case *files.FolderMetadata:
			o = s.formatFolderObject(meta)
		case *files.FileMetadata:
			o = s.formatFileObject(meta)
		}

		page.Data = append(page.Data, o)
	}

	if !output.HasMore {
		return IterateDone
	}

	input.cursor = output.Cursor
	return nil
}

func (s *Storage) read(ctx context.Context, path string, w io.Writer, opt pairStorageRead) (n int64, err error) {
	rp := s.getAbsPath(path)

	input := &files.DownloadArg{
		Path: rp,
	}

	_, rc, err := s.client.Download(input)
	if err != nil {
		return 0, err
	}

	if opt.HasSize {
		rc = iowrap.LimitReadCloser(rc, opt.Size)
	}

	if opt.HasIoCallback {
		rc = iowrap.CallbackReadCloser(rc, opt.IoCallback)
	}

	return io.Copy(w, rc)
}

func (s *Storage) stat(ctx context.Context, path string, opt pairStorageStat) (o *Object, err error) {
	rp := s.getAbsPath(path)

	input := &files.GetMetadataArg{
		Path: rp,
	}

	output, err := s.client.GetMetadata(input)
	if err != nil {
		return nil, err
	}

	switch meta := output.(type) {
	case *files.FolderMetadata:
		o = s.formatFolderObject(meta)
	case *files.FileMetadata:
		o = s.formatFileObject(meta)
	}

	return o, nil
}

func (s *Storage) write(ctx context.Context, path string, r io.Reader, size int64, opt pairStorageWrite) (n int64, err error) {
	rp := s.getAbsPath(path)

	r = io.LimitReader(r, size)

	if opt.HasIoCallback {
		r = iowrap.CallbackReader(r, opt.IoCallback)
	}

	input := &files.CommitInfo{
		Path: rp,
		Mode: &files.WriteMode{
			Tagged: dropbox.Tagged{
				Tag: files.WriteModeAdd,
			},
		},
	}

	_, err = s.client.Upload(input, r)
	if err != nil {
		return 0, err
	}

	return size, nil
}

func (s *Storage) writeAppend(ctx context.Context, o *Object, r io.Reader, size int64, opt pairStorageWriteAppend) (n int64, err error) {
	if !o.Mode.IsAppend() {
		err = fmt.Errorf("object not appendable")
		return
	}

	sessionId := GetObjectMetadata(o).UploadSessionID

	offset, ok := o.GetAppendOffset()
	if !ok {
		err = fmt.Errorf("append offset is not set")
		return
	}

	cursor := &files.UploadSessionCursor{
		SessionId: sessionId,
		Offset:    uint64(offset),
	}

	appendArg := &files.UploadSessionAppendArg{
		Cursor: cursor,
		Close:  false,
	}

	err = s.client.UploadSessionAppendV2(appendArg, r)
	if err != nil {
		return
	}

	offset += size
	o.SetAppendOffset(offset)

	return size, nil
}
