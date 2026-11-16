package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/forgen-team/punisher/internal/backup"
)

// Export handles GET /v1/export — the data folder as a zip download
// (punisher.db copy + manifest.json; same as `punisher export`).
func (h *StatusHandler) Export(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if _, err := backup.Export(r.Context(), filepath.Join(h.DataDir, "punisher.db"), h.AppVersion, &buf); err != nil {
		log.Printf("export: %v", err)
		writeError(w, r, http.StatusInternalServerError, ErrorInternal, "export failed")
		return
	}
	name := fmt.Sprintf("punisher-backup-%s.zip", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(buf.Bytes())
}
