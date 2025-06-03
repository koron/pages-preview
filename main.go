package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	httpAddr         = "localhost:8080"
	archiveNameOuter = "github-pages.zip"
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

func (ztr *ZipTarReader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path
	if data, ok := ztr.files[name]; ok {
		http.ServeContent(w, r, name, time.Now(), bytes.NewReader(data))
		return
	}
	// redirect to directory if available
	if !strings.HasSuffix(name, "/") {
		if _, ok := ztr.dirs[name+"/"]; !ok {
			http.NotFound(w, r)
			return
		}
		r.URL.Path += "/"
		http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
		return
	}
	// serve index.html
	name += "index.html"
	if data, ok := ztr.files[name]; ok {
		http.ServeContent(w, r, name, time.Now(), bytes.NewReader(data))
		return
	}
	http.NotFound(w, r)
}

func main() {
	flag.StringVar(&httpAddr, "addr", httpAddr, `HTTP server listen address`)
	flag.StringVar(&archiveNameOuter, "outer", archiveNameOuter, `name of outer archive`)
	flag.StringVar(&archiveNameInner, "inner", archiveNameInner, `name of inner archive`)
	flag.Parse()

	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		log.Fatalf("invalid addr: %s", err)
	}
	if host == "" {
		host = "localhost"
	}
	addr := net.JoinHostPort(host, port)

	handler, err := NewReader(archiveNameOuter, archiveNameInner)
	if err != nil {
		log.Fatalf("failed to open the archive: %s", err)
	}

	log.Printf("hosting %s now. please open http://%s/ with your browser", archiveNameOuter, addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
