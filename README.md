# feedfinder

Discover RSS, Atom, and JSON feeds from any URL.

## Install

```bash
go install github.com/ramadhantriyant/feedfinder/cmd/main@latest
```

Or build from source:

```bash
git clone https://github.com/ramadhantriyant/feedfinder
cd feedfinder
go build -o feedfinder ./cmd/main
```

## Usage

```
feedfinder [flags] <url>
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `-timeout` | `15` | HTTP timeout in seconds |
| `-ua` | | Custom `User-Agent` header |
| `-v` | | Verbose debug output |
| `-no-color` | | Disable coloured output |

**Examples**

```bash
feedfinder https://blog.rust-lang.org
feedfinder -v -timeout 20 news.ycombinator.com
feedfinder -ua "Mozilla/5.0" https://example.com
```

## How it works

Feed discovery runs in four stages, stopping as soon as feeds are found:

1. **Content-Type header** — if the URL itself serves `application/rss+xml`, `application/atom+xml`, or `application/feed+json`, it is returned immediately.
2. **HTML `<link>` tags** — scans for `<link rel="alternate">` and `<link rel="self">` tags with a feed MIME type.
3. **Common path probing** — tries well-known paths (`/feed`, `/rss`, `/atom.xml`, `/index.xml`, etc.) and validates each response.

## Dependencies

None. feedfinder uses only the Go standard library.
