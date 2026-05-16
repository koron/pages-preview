# pages-preview/pages-preview

[![PkgGoDev](https://pkg.go.dev/badge/github.com/koron/pages-preview)](https://pkg.go.dev/github.com/koron/pages-preview)
[![Actions/Go](https://github.com/koron/pages-preview/workflows/Go/badge.svg)](https://github.com/koron/pages-preview/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron/pages-preview)](https://goreportcard.com/report/github.com/koron/pages-preview)

A local web server that previews the github-pages.zip artifact created by [actions/upload-pages-artifact](https://github.com/marketplace/actions/upload-github-pages-artifact).

-   Preview the downloaded github-pages.zip file without extracting it.
-   Preview the github-pages.zip artifact by specifying the URL of the action run (PAT required).

## Install and update

```console
$ go install github.com/koron/pages-preview@latest
```

Or download pre-compiled binary from [latest](https://github.com/koron/pages-preview/releases/latest).

## Getting started

Running the following command will allow you to preview the downloaded ./github-pages.zip at <http://localhost:8080/>. 

```console
$ pages-preview
```

Specifying the URL for running the action using `upload-pages-artifact`, as shown in the following command, will temporarily and implicitly download the artifact `github-pages.zip` and allow you to preview it at `<http://localhost:8080/>`.

At this time, you need to set the environment variable `PAGES_PREVIEW_GITHUB_TOKEN` to the [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) you created beforehand.

```console
$ pages-preview https://github.com/vim-jp/vim-jp.github.io/actions/runs/25962415678
```
