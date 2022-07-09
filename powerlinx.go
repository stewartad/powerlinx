package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var markdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Typographer,
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithUnsafe(),
	),
)

// A Site holds all the information about this website
type Site struct {
	content     fs.FS
	templates   fs.FS
	PageMap     map[string]*Page
	Views       map[string]*View
	SortedPages []*Page
}

// NewSite creates a new Site and takes two fs.FS parameters
// content is a FS containing  all your content
// templates is a FS containing HTML templates,
// it's root directory should contain layouts for individual pages
// and a base/ directory containing layouts for the whole site
// NewSite automatically loads all contents of data into memory at startup,
// which makes this unfit for exceptionally large sites
func NewSite(content, templates fs.FS) *Site {
	site := Site{
		content:   content,
		templates: templates,
		PageMap:   make(map[string]*Page),
		Views:     make(map[string]*View),
	}
	err := site.loadAllStaticPages()
	if err != nil {
		log.Fatal(err)
	}
	site.SortedPages = site.sortPages()
	return &site
}

// AddView adds a view to the site's internal map
func (s *Site) AddView(name string, v *View) {
	s.Views[name] = v
}

// A Page contains metadata and content for a single webpages
// Metadata is standard json surrounded by "---"
type Page struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	Type      string    `json:"type"`
	Body      template.HTML
	Url       string
}

func (s *Site) loadAllStaticPages() error {
	err := fs.WalkDir(s.content, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		} else {
			page, ferr := s.parseSinglePage(path)
			if ferr != nil {
				return ferr
			}
			s.PageMap[page.Url] = page
		}
		return nil
	})
	return err
}

func (s *Site) sortPages() []*Page {
	all := make([]*Page, 0, len(s.PageMap))
	for _, value := range s.PageMap {
		all = append(all, value)
	}
	sort.Sort(byTime(all))
	return all
}

type byTime []*Page

func (t byTime) Len() int {
	return len(t)
}

func (t byTime) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t byTime) Less(i, j int) bool {
	return t[j].CreatedAt.Before(t[i].CreatedAt)
}

type body []byte
type metadata []byte

// GetRecentPages returns a slice of Pages of the specified pageType,
// as defined in a Page's metadata.
// If pageType is "", Pages of any type are returned
// The length of the returned slice is either count or the length of s.PageMap,
// whichever is smaller
func (s *Site) GetRecentPages(count int, pageType string) []*Page {
	all := make([]*Page, 0, count)

	for _, page := range s.SortedPages {
		if page.Type == pageType || pageType == "" {
			all = append(all, page)
		}
		if len(all) == count {
			break
		}
	}
	return all
}

func (s *Site) parseSinglePage(path string) (*Page, error) {
	file, err := s.content.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metadata, body := parseMetadata(file)
	filetype := filepath.Ext(path)

	var bodyHTML template.HTML
	url := strings.TrimSuffix("/"+path, filetype)

	// if md, parse to html
	// if html, parse as-is
	if filetype == ".md" {
		var buf bytes.Buffer
		if err := markdown.Convert(body, &buf); err != nil {
			log.Panic(err)
		}
		bodyHTML = template.HTML(buf.String())

	} else if filetype == ".html" {
		bodyHTML = template.HTML(string(body))
	}

	page := &Page{Body: bodyHTML, Url: url}
	log.Printf("Loading Page %s, Url %s", path, url)
	// parse metadata
	if len(metadata) > 0 {
		err := json.Unmarshal(metadata, page)
		if err != nil {
			return nil, err
		}
	}
	return page, nil
}

// parseMetadata parses JSON metadata at the top of the file, surrounded by "---"
func parseMetadata(r io.Reader) (metadata, body) {
	scanner := bufio.NewScanner(r)
	metadata := []byte{}
	body := []byte{}
	// separate metadata and content
	count := 0 // counter for metadata delimiter, expecting either zero or two
	for scanner.Scan() {
		if scanner.Text() == "---" {
			count++
			continue
		}
		if 0 < count && count < 2 {
			metadataBytes := scanner.Bytes()
			metadata = append(metadata, metadataBytes...)
		} else {
			contentBytes := scanner.Bytes()
			body = append(body, contentBytes...)
			body = append(body, '\n')
		}
	}
	return metadata, body
}

// View stores information about a template
type View struct {
	Template *template.Template
	Layout   string
}

// Execute a given template, passing it data
func (v *View) Render(w http.ResponseWriter, data interface{}) error {
	return v.Template.ExecuteTemplate(w, v.Layout, data)
}

// Add a View to the Site
func (s *Site) NewView(layout string, lastTmpl string) *View {
	t, err := template.ParseFS(s.templates, lastTmpl, "base/*.html")
	if err != nil {
		log.Fatal(err)
	}

	return &View{Template: t, Layout: layout}
}
