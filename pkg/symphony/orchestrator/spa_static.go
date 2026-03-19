package orchestrator

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

func spaHandler(staticFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		name := "index.html"
		if p := strings.TrimSpace(r.URL.Path); p != "" && p != "/" {
			clean := path.Clean(p)
			clean = strings.TrimPrefix(clean, "/")
			if clean != "" && clean != "." {
				if _, err := fs.Stat(staticFS, clean); err == nil {
					name = clean
				}
			}
		}

		b, err := fs.ReadFile(staticFS, name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	})
}

