package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/koron-go/ctxsrv"
	"github.com/koron/pages-preview/internal/github"
)

const defaultName = "github-pages.zip"

var (
	httpAddr         = "localhost:8080"
	archiveNameOuter string
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

var actionURLMx = regexp.MustCompile(`https://github.com/([^/]+)/([^/]+)/actions/runs/(\d+)`)

// downloadActionArtifactTemporary downloads github-pages.zip directly from GHA artifacts.
func downloadActionArtifactTemporary(actionURL, artifactName string) (tmpDir, outerName string, err error) {
	m := actionURLMx.FindStringSubmatch(actionURL)
	if m == nil {
		return "", "", fmt.Errorf("invalid actions URL: %s", actionURL)
	}
	owner, repo, runID := m[1], m[2], m[3]
	a, err := github.GetArtifact(context.Background(), owner, repo, runID, artifactName)
	if err != nil {
		return "", "", err
	}

	tmpdir, err := os.MkdirTemp("", "pages-pareview")
	if err != nil {
		return "", "", err
	}
	name := filepath.Join(tmpdir, defaultName)
	log.Printf("downloading an articat to: %s", name)

	// TODO: show the progress of the download
	err = a.Download(context.Background(), name)
	if err != nil {
		defer os.RemoveAll(tmpdir)
		return "", "", err
	}

	return tmpdir, name, nil
}

func run() error {
	flag.StringVar(&httpAddr, "addr", httpAddr, `HTTP server listen address`)
	flag.StringVar(&archiveNameOuter, "outer", defaultName, `name of outer archive`)
	flag.StringVar(&archiveNameInner, "inner", archiveNameInner, `name of inner archive`)
	flag.Parse()

	host, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return fmt.Errorf("invalid addr: %w", err)
	}
	if host == "" {
		host = "localhost"
	}
	addr := net.JoinHostPort(host, port)

	if flag.NArg() > 0 {
		if flag.NArg() >= 2 {
			return errors.New("too many arguments. required zero or one")
		}
		tmpDir, outerName, err := downloadActionArtifactTemporary(flag.Arg(0), archiveNameOuter)
		if err != nil {
			return fmt.Errorf("failed to download the artifact from the action: %w", err)
		}
		if tmpDir != "" {
			defer os.RemoveAll(tmpDir)
		}
		archiveNameOuter = outerName
	}

	handler, err := NewReader(archiveNameOuter, archiveNameInner)
	if err != nil {
		return fmt.Errorf("failed to open the archive: %w", err)
	}

	log.Printf("hosting %s now. please open http://%s/ with your browser", archiveNameOuter, addr)
	return listenAndServe(addr, handler)
}

func listenAndServe(addr string, handler http.Handler) error {
	// Capture Ctrl-C to ensure tmpdir is deleted
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return ctxsrv.HTTP(&http.Server{Addr: addr, Handler: handler}).ServeWithContext(ctx)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
