package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

var (
	httpAddr         = "localhost:8080"
	archiveNameOuter = "githut-pages.zip"
	archiveNameInner = "artifact.tar"
)

type ZipTarReader struct {
	dirs  map[string]struct{}
	files map[string][]byte
}

func NewReader(outer, inner string) (*ZipTarReader, error) {
	zr, err := zip.OpenReader(outer)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	fr, err := zr.Open(inner)
	if err != nil {
		return nil, err
	}
	_ = fr
	files := map[string][]byte{}
	dirs := map[string]struct{}{}
	tr := tar.NewReader(fr)
	for {
		h, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		name, _ := strings.CutPrefix(h.Name, ".")
		switch h.Typeflag {
		case tar.TypeReg:
			// load a regular file.
			data := make([]byte, h.Size)
			_, err = io.ReadFull(tr, data)
			if err != nil {
				return nil, err
			}
			files[name] = data

		case tar.TypeDir:
			dirs[name] = struct{}{}
		}
	}
	return &ZipTarReader{
		dirs:  dirs,
		files: files,
	}, nil
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }
func (fi *fileInfo) IsDir() bool        { return fi.isDir }
func (fi *fileInfo) Sys() any           { return fi.sys }

var (
	errMissingRead    = errors.New("io.File directory missing Read method")
	errMissingSeek    = errors.New("io.File directory missing Seek method")
	errMissingReadDir = errors.New("io.File directory missing ReadDir method")
)

type memFile struct {
	fi fileInfo
	br *bytes.Reader
}

func (f *memFile) Close() error { return nil }

func (f *memFile) Read(b []byte) (int, error) {
	return f.br.Read(b)
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	return f.br.Seek(offset, whence)
}

func (f *memFile) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, errMissingReadDir
}

func (f *memFile) Stat() (fs.FileInfo, error) { return &f.fi, nil }

type memDir struct {
	fi fileInfo
}

func (d *memDir) Close() error { return nil }

func (d *memDir) Read(b []byte) (int, error) {
	return 0, errMissingRead
}

func (d *memDir) Seek(offset int64, whence int) (int64, error) {
	return 0, errMissingSeek
}

func (d *memDir) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, errMissingReadDir
}

func (d *memDir) Stat() (fs.FileInfo, error) { return &d.fi, nil }

func (ztr *ZipTarReader) Open(name string) (http.File, error) {
	if _, ok := ztr.dirs[name]; ok {
		return &memDir{
			fi: fileInfo{
				name:    name,
				size:    0,
				mode:    0555,
				modTime: time.Now(),
				isDir:   true,
				sys:     nil,
			},
		}, nil
	}
	data, ok := ztr.files[name]
	if !ok {
		name = path.Join(name, "index.html")
		data, ok = ztr.files[name]
		if !ok {
			return nil, fs.ErrNotExist
		}
	}
	return &memFile{
		fi: fileInfo{
			name:    name,
			size:    int64(len(data)),
			mode:    0666,
			modTime: time.Now(),
			isDir:   false,
			sys:     nil,
		},
		br: bytes.NewReader(data),
	}, nil
}

func archiveToFS(outer, inner string) (http.FileSystem, error) {
	return NewReader(outer, inner)
}

func main() {
	flag.StringVar(&httpAddr, "addr", httpAddr, `HTTP server listen address`)
	flag.StringVar(&archiveNameOuter, "outer", archiveNameOuter, `name of outer archive`)
	flag.StringVar(&archiveNameInner, "inner", archiveNameInner, `name of inner archive`)
	flag.Parse()

	fsys, err := archiveToFS(archiveNameOuter, archiveNameInner)
	if err != nil {
		log.Fatalf("failed to open the archive: %s", err)
	}
	handler := http.FileServer(fsys)
	log.Printf("listening on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, handler); err != nil {
		log.Fatal(err)
	}
}
