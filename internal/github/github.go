// Package github wraps how to access GitHub.
package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/koron-go/jsonhttpc"
)

var defaultClient *jsonhttpc.Client

func Header(h http.Header) http.Header {
	if h == nil {
		h = make(http.Header)
	}
	h.Set("Accept", "application/vnd.github+json")
	h.Set("X-GitHub-Api-Version", "2022-11-28")
	// FIXME: Improved the way to obtain tokens
	if s := os.Getenv("GITHUB_TOKEN"); s != "" {
		h.Set("Authorization", "Bearer "+s)
	}
	return h
}

func init() {
	defaultClient = jsonhttpc.New(nil)
	defaultClient.WithHeader(Header(nil))
}

type Artifact struct {
	ID                 int       `json:"id"`
	NodeID             string    `json:"node_id"`
	Name               string    `json:"name"`
	SizeInBytes        int       `json:"size_in_bytes"`
	URL                string    `json:"url"`
	ArchiveDownloadURL string    `json:"archive_download_url"`
	Expired            bool      `json:"expired"`
	Digest             string    `json:"digest"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

type ArtifactList struct {
	TotalCount int         `json:"total_count"`
	Artifacts  []*Artifact `json:"artifacts"`
}

var ErrNoArtifactsFound = errors.New("no artifacts found")

func GetArtifact(ctx context.Context, owner, repo, runID, artifactName string) (*Artifact, error) {
	name, ext := artifactName, path.Ext(artifactName)
	if len(ext) > 0 {
		name = name[:len(name)-len(ext)]
		ext = "/" + strings.TrimLeft(ext, ".")
	}
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%s/artifacts", owner, repo, runID)
	var list ArtifactList
	err := defaultClient.Do(ctx, "GET", u, nil, &list)
	if err != nil {
		return nil, err
	}
	for _, a := range list.Artifacts {
		if !a.Expired && a.Name == name && strings.HasSuffix(a.ArchiveDownloadURL, ext) {
			return a, nil
		}
	}
	return nil, ErrNoArtifactsFound
}

func (a *Artifact) Download(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.ArchiveDownloadURL, nil)
	if err != nil {
		return err
	}
	Header(req.Header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download request failed: status=%d", resp.StatusCode)
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
