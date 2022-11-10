package powerlinx

import (
	"html/template"
	"io/fs"
	"log"
	"path"
	"path/filepath"
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

// A Page contains metadata and content for a single webpages
// Metadata is standard json surrounded by "---"
type Page struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	Type      string    `json:"type"`
	Url       string
	Body      template.HTML
	View      *View
}

// A Site holds all the information about this website
type Site struct {
	content     fs.FS
	templates   fs.FS
	PageMap     map[string]*Page
	Views       map[string]*View
	StaticViews map[string]*View
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
		content:     content,
		templates:   templates,
		PageMap:     make(map[string]*Page),
		Views:       make(map[string]*View),
		StaticViews: make(map[string]*View),
	}
	err := site.createViewsFromTemplates()
	if err != nil {
		log.Fatal(err)
	}
	err = site.discoverPages()
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

// looks for _index.html and _single.html
//
func (s *Site) createViewsFromTemplates() error {
	// walk template directory
	err := fs.WalkDir(s.templates, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalln(err)
		}
		// skip base templates
		if d.IsDir() || strings.Contains(filePath, "base") || filePath == "." {
			return nil
		}
		if d.Name() == "_index.html" {
			viewName := path.Dir(filePath)
			if viewName == "." {
				viewName = "/"
			} else {
				viewName = "/" + viewName
			}
			s.StaticViews[viewName] = s.NewView("layout.html", path.Clean(filePath))
		} else if d.Name() == "_single.html" {
			// TODO: remove hardcoding of _single.html to allow for custom templates per-page
			// _single.html will be used by default if no other template is found
			viewName := path.Dir(filePath)
			if viewName == "." {
				viewName = "/page"
			} else {
				viewName = "/" + viewName + "/page"
			}
			s.StaticViews[viewName] = s.NewView("layout.html", path.Clean(filePath))
		}
		return nil
	})
	return err
}

func (s *Site) discoverPages() error {
	err := fs.WalkDir(s.content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil
		} else {
			page, ferr := s.createPageFromFile(path)
			if ferr != nil {
				return ferr
			}
			s.PageMap[page.Url] = page
		}
		return nil
	})
	return err
}

func (s *Site) createPageFromFile(filePath string) (*Page, error) {
	file, err := s.content.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metadata, body := separateMetadata(file)
	filetype := filepath.Ext(filePath)

	page, err := parseMetadata(metadata)
	if err != nil {
		return nil, err
	}

	page.Body = convertToHTML(body, filetype)
	page.Url = strings.TrimSuffix("/"+filePath, filetype)

	tmplName := "page"
	if path.Base(filePath) == "index.html" {
		tmplName = ""
	}

	page.View = s.getView(path.Dir(page.Url), tmplName)

	log.Printf("Loading Page %s, Url %s", filePath, page.Url)

	return page, nil
}

func (s *Site) getView(pageDir string, templateName string) *View {

	currDir := pageDir
	for currDir != "." {
		tmpl := path.Clean(currDir + "/" + templateName)
		view, exists := s.StaticViews[tmpl]
		if exists {
			return view
		}
		currDir = path.Dir(currDir)
	}
	return s.StaticViews["/"+templateName]
}
