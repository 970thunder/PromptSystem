package api

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// handleStaticUpload serves local upload files with explicit, safe headers and
// without directory listing or path traversal. The request still passes through
// the global chain (so X-Content-Type-Options: nosniff is set), but this handler
// adds the correct Content-Type and a long cache policy for immutable objects.
func (s *server) handleStaticUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeMethodNotAllowed(w)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/uploads/")
	name = strings.Trim(name, "/")
	if name == "" {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	// Normalize and reject any traversal attempt before joining with the base
	// directory. path.Clean collapses ".." segments; a leading ".." after
	// cleaning means the request tried to escape the upload root.
	cleaned := path.Clean(name)
	if strings.HasPrefix(cleaned, "..") {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}
	if strings.Contains(cleaned, "..") {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	fullPath := filepath.Join(s.config.UploadDir, cleaned)

	// Defense in depth: confirm the resolved absolute path is still inside the
	// configured upload directory.
	base, err := filepath.Abs(s.config.UploadDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, Message: "Internal server error"})
		return
	}
	abs, err := filepath.Abs(fullPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}
	if abs != base && !strings.HasPrefix(abs, base+string(os.PathSeparator)) {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		// Never list a directory; treat missing or directory paths as 404.
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}

	file, err := os.Open(abs)
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiResponse[any]{Code: 404, Message: "Not found"})
		return
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(abs))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Uploaded images are content-addressed by random key, so they are safe to
	// cache for a long time.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
