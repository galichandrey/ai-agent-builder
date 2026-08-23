// Package templates works with LangFlow's official template format: opaque
// JSON files (as shipped in langflow/initial_setup/starter_projects/) plus a
// small typed catalog surface. Raw-first: file bytes are preserved verbatim;
// only catalog fields (name/description/tags) are decoded.
package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File is a parsed native-format template.
type File struct {
	Name         string
	Description  string
	EndpointName string
	Tags         []string
	IsComponent  bool
	NodeCount    int
	Dir          string // "native", "custom" or "gallery"
	Category     string // gallery only: first path segment under gallery/
	Subcategory  string // gallery only: second path segment ("" if none)
	Path         string

	Raw     json.RawMessage // whole template file, verbatim
	dataRaw json.RawMessage // cached .data payload
}

// Parse decodes the catalog fields of a template file at path.
func Parse(path, dir string) (*File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var head struct {
		Name         *string  `json:"name"`
		Description  string   `json:"description"`
		EndpointName *string  `json:"endpoint_name"`
		Tags         []string `json:"tags"`
		IsComponent  bool     `json:"is_component"`
		Data         struct {
			Nodes []json.RawMessage `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	name := ""
	if head.Name != nil {
		name = *head.Name
	}
	if name == "" {
		base := filepath.Base(path)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	ep := ""
	if head.EndpointName != nil {
		ep = *head.EndpointName
	}
	return &File{
		Name:         name,
		Description:  head.Description,
		EndpointName: ep,
		Tags:         head.Tags,
		IsComponent:  head.IsComponent,
		NodeCount:    len(head.Data.Nodes),
		Dir:          dir,
		Path:         path,
		Raw:          raw,
	}, nil
}

// dirRank orders sources: native starters, then custom, then gallery.
func dirRank(d string) int {
	switch d {
	case "native":
		return 0
	case "custom":
		return 1
	default:
		return 2
	}
}

// galleryCategory derives Category/Subcategory from a path under gallery/
// ("gallery/business/sales_marketing/x.json" -> business, sales_marketing).
func galleryCategory(root, path string) (cat, sub string) {
	rel, err := filepath.Rel(filepath.Join(root, "gallery"), path)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) >= 2 {
		cat = parts[0]
	}
	if len(parts) >= 3 {
		sub = parts[1]
	}
	return cat, sub
}

// looksLikeTemplate reports whether raw JSON is a template object (must have
// a "data" payload). Non-template files (ingest manifests, stray arrays) are
// skipped by LoadDir instead of failing the whole directory scan.
func looksLikeTemplate(raw []byte) bool {
	var head struct {
		Data *json.RawMessage `json:"data"`
	}
	return json.Unmarshal(raw, &head) == nil && head.Data != nil
}

// LoadDir scans root/native and root/custom (flat) plus root/gallery
// recursively (subfolders = category/subcategory) for *.json templates.
func LoadDir(root string) ([]*File, error) {
	var files []*File
	for _, sub := range []string{"native", "custom"} {
		dir := filepath.Join(root, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			f, err := Parse(filepath.Join(dir, e.Name()), sub)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
	}
	galleryRoot := filepath.Join(root, "gallery")
	err := filepath.WalkDir(galleryRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !looksLikeTemplate(raw) {
			return nil // manifests and other non-template files are skipped
		}
		f, err := Parse(path, "gallery")
		if err != nil {
			return err
		}
		f.Category, f.Subcategory = galleryCategory(root, path)
		files = append(files, f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Dir != files[j].Dir {
			return dirRank(files[i].Dir) < dirRank(files[j].Dir)
		}
		return files[i].Name < files[j].Name
	})
	return files, nil
}

// Lookup finds a template by exact/case-insensitive Name or filename slug.
func Lookup(files []*File, query string) (*File, bool) {
	slug := Slugify(query)
	for _, f := range files {
		if f.Name == query || strings.EqualFold(f.Name, query) || Slugify(f.Name) == slug {
			return f, true
		}
	}
	return nil, false
}

// MatchesQuery reports whether a template matches a free-text query:
// every whitespace-separated token must appear (case-insensitively) in the
// name, description, or tags. An empty query matches everything.
func MatchesQuery(f *File, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	hay := strings.ToLower(f.Name + " " + f.Description + " " + strings.Join(f.Tags, " "))
	for _, tok := range strings.Fields(query) {
		if !strings.Contains(hay, tok) {
			return false
		}
	}
	return true
}

// DataRaw returns the template's .data payload as raw JSON.
func (f *File) DataRaw() (json.RawMessage, error) {
	if f.dataRaw != nil {
		return f.dataRaw, nil
	}
	var head struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(f.Raw, &head); err != nil {
		return nil, err
	}
	f.dataRaw = head.Data
	return f.dataRaw, nil
}

// Slugify converts a display name to a filesystem-friendly slug:
// "Simple Agent" -> "simple_agent".
func Slugify(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
