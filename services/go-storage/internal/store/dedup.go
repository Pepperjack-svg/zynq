package store

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// dedupMIMEs lists MIME types that qualify for content-addressable deduplication.
//
// Rationale: these types are commonly re-uploaded unchanged by multiple users
// (team-shared PDFs, logos, diagrams). Storing them in the CAS saves significant
// disk for collaborative platforms.  Binary types (video, zip, executables) are
// typically unique per upload and would produce dedup misses at the cost of an
// extra disk write, so they are excluded.
var dedupMIMEs = map[string]bool{
	"text/plain":      true, // .txt
	"application/pdf": true, // .pdf
	"image/jpeg":      true, // .jpg / .jpeg
	"image/png":       true, // .png
	"image/svg+xml":   true, // .svg  (browsers sniff as text/xml or image/svg+xml)
	"text/xml":        true, // .svg when served with XML declaration
	"image/gif":       true, // .gif
	"image/webp":      true, // .webp
}

// dedupOOXML lists file extensions whose content is a ZIP archive but which
// represent deduplicated document formats.  net/http.DetectContentType cannot
// distinguish OOXML from a generic ZIP, so the extension is used as a safe
// secondary signal.
//
// IMPORTANT: this map is only consulted when the MIME sniffer already returned
// "application/zip" or "application/octet-stream" — a positive MIME detection
// always takes precedence.
var dedupOOXML = map[string]bool{
	".docx": true, // Word
	".xlsx": true, // Excel
	".pptx": true, // PowerPoint
	".odt":  true, // LibreOffice Writer
	".ods":  true, // LibreOffice Calc
	".odp":  true, // LibreOffice Impress
}

// ShouldDedup reads up to 512 bytes from r to detect the MIME type, then
// returns:
//
//   - dedupable: whether this content should be routed through the CAS
//   - full: an io.Reader that replays the sniffed bytes followed by the rest of r
//
// fileName is the original client-supplied filename (from X-File-Name header).
// It is used ONLY as a fallback for OOXML formats that are byte-identical to
// generic ZIP archives.  Never use fileName to override a positive MIME match.
//
// The first 512 bytes are never written to disk at this stage — they are held
// in a small heap buffer and prepended to r via io.MultiReader, so the full
// body is available to the caller without re-reading.
func ShouldDedup(r io.Reader, fileName string) (dedupable bool, full io.Reader) {
	sniff := make([]byte, 512)
	n, _ := io.ReadFull(r, sniff)
	sniff = sniff[:n]

	// Reconstruct the full stream: sniffed prefix + remainder.
	full = io.MultiReader(bytes.NewReader(sniff), r)

	if n == 0 {
		// Empty body — store normally; nothing to dedup.
		return false, full
	}

	// net/http.DetectContentType is the same sniffer used by browsers.
	// It inspects up to 512 bytes and never reads from the network.
	mime := http.DetectContentType(sniff)

	// Strip MIME parameters: "text/plain; charset=utf-8" → "text/plain"
	if i := strings.IndexByte(mime, ';'); i != -1 {
		mime = strings.TrimSpace(mime[:i])
	}

	if dedupMIMEs[mime] {
		return true, full
	}

	// OOXML / ODF fallback: these are ZIP archives at the byte level.
	// Only dedup when the extension confirms the document type.
	if mime == "application/zip" || mime == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(fileName))
		if dedupOOXML[ext] {
			return true, full
		}
	}

	return false, full
}
