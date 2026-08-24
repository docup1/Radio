package infra

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler serves a built single-page application from dir. It implements
// SPA fallback: any GET that does not resolve to a real file (and is not claimed
// by a more specific route such as /api or /swagger) is served index.html so
// client-side routes keep working on direct hit / refresh.
//
// All paths are resolved against an absolute base directory; any request that
// would escape it (path traversal) is rejected, and files are served with a
// cleaned request path to avoid spurious redirects.
func StaticHandler(dir string) http.Handler {
	base, err := filepath.Abs(dir)
	if err != nil {
		base = filepath.Clean(dir)
	}
	sep := string(os.PathSeparator)

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		index := filepath.Join(base, "index.html")
		if _, err := os.Stat(index); err != nil {
			http.NotFound(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r2, index)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		clean := filepath.Clean(r.URL.Path)
		if clean == "/" || clean == "." {
			serveIndex(w, r)
			return
		}

		target := filepath.Join(base, clean)
		if target != base && !strings.HasPrefix(target, base+sep) {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(target)
		if err != nil {
			serveIndex(w, r)
			return
		}
		if info.IsDir() {
			serveIndex(w, r)
			return
		}

		rel, rerr := filepath.Rel(base, target)
		if rerr != nil {
			http.NotFound(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + filepath.ToSlash(rel)
		http.ServeFile(w, r2, target)
	})
}
