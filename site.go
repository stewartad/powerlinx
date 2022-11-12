package powerlinx

import (
	"errors"
	"io/fs"
	"log"
	"path"
	"strings"

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
	content      fs.FS
	templates    fs.FS
	PageMap      map[string]*DetailPage
	ListPageMap  map[string]*ListPage
	DynamicViews map[string]*View
	StaticViews  map[string]*View
	SortedPages  []*DetailPage
	Config       *SiteConfig
}

// NewSite creates a new Site and takes two fs.FS parameters
// content is a FS containing  all your content
// templates is a FS containing HTML templates,
// it's root directory should contain layouts for individual pages
// and a base/ directory containing layouts for the whole site
func NewSite(content, templates fs.FS, opts ...SiteOption) *Site {
	c := NewConfig()
	for _, opt := range opts {
		opt.SetSiteOption(c)
	}
	return &Site{
		content:      content,
		templates:    templates,
		PageMap:      make(map[string]*DetailPage),
		ListPageMap:  make(map[string]*ListPage),
		DynamicViews: make(map[string]*View),
		StaticViews:  make(map[string]*View),
		Config:       c,
	}
}

// Build will discover templates, create views for the templates, then discover content pages to be parsed, then generate aggregate pages
func (s *Site) Build() {
	err := s.createViewsFromTemplates()
	if err != nil {
		log.Fatal(err)
	}
	err = s.discoverPages()
	if err != nil {
		log.Fatal(err)
	}
	err = s.discoverListPages()
	if err != nil {
		log.Fatal(err)
	}
	s.SortedPages = s.sortAllPages()
}

// AddView adds a view to the site's internal map
func (s *Site) AddView(name string, v *View) {
	s.DynamicViews[name] = v
}

// GetRecentPages returns a slice of Pages of the specified pageType,
// as defined in a Page's metadata.
// If pageType is "", Pages of any type are returned
// The length of the returned slice is either count or the length of s.PageMap,
// whichever is smaller
func (s *Site) GetRecentPages(count int, pageType string) []*DetailPage {
	all := make([]*DetailPage, 0, count)

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
		if d.IsDir() || strings.Contains(filePath, "base") {
			return nil
		}

		tmplType := strings.TrimPrefix(strings.TrimSuffix(path.Base(filePath), ".html"), "_")
		viewName := path.Clean(strings.TrimPrefix(path.Dir(filePath)+"/"+tmplType, "./"))
		s.StaticViews[viewName] = s.NewView("layout.html", path.Clean(filePath))

		return nil
	})
	return err
}

func (s *Site) discoverPages() error {
	err := fs.WalkDir(s.content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.Name() == "." || d.IsDir() {
			// skipping directories, will discover them in the next pass
			return nil
		}

		page, ferr := s.createPageFromFile(path)
		if ferr != nil {
			return ferr
		}
		if !page.Draft || s.Config.IncludeDrafts {
			s.PageMap[page.Url] = page
		}

		return nil
	})

	return err
}

func (s *Site) discoverListPages() error {
	err := fs.WalkDir(s.content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.Name() == "." || !d.IsDir() {
			return nil
		}

		page, ferr := s.createListPage(path, d.Name())
		if ferr != nil {
			return ferr
		}
		s.ListPageMap[page.Url] = page
		return nil
	})
	return err
}

func (s *Site) createPageFromFile(filePath string) (*DetailPage, error) {
	file, err := s.content.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	page, err := NewDetailPage(file, filePath)
	if err != nil {
		return nil, err
	}

	if path.Base(filePath) == "index.html" {
		page.View = s.getView(path.Dir(filePath), TMPL_INDEX)
	} else {
		page.View = s.getView(path.Dir(filePath), TMPL_PAGE)
	}

	log.Printf("Loading Page %s, Url %s", filePath, page.Url)

	return page, nil
}

func (s *Site) getView(pageDir string, templateName string) *View {

	currDir := pageDir
	for currDir != "." && currDir != "/" {
		tmpl := path.Clean(currDir + "/" + templateName)
		view, exists := s.StaticViews[tmpl]
		if exists {
			return view
		}
		currDir = path.Dir(currDir)
	}
	// Hack to prevent homepage from rendering as a ListPage
	return s.StaticViews[templateName]
}

func (s *Site) getAllPagesInDir(dir string) []*DetailPage {
	// TODO: check whether to include subdirs or not
	pages := make([]*DetailPage, 0)
	for url, page := range s.PageMap {
		if strings.HasPrefix(url, dir) {
			pages = append(pages, page)
		}
	}
	return pages
}

func (s *Site) createListPage(dir string, title string) (*ListPage, error) {

	view := s.getView(dir, TMPL_LIST)
	if view == nil {
		log.Printf("could not find view")
		// TODO: error
		return nil, errors.New("could not find view")
	}
	pages := s.getAllPagesInDir(path.Clean("/" + dir))
	sortPageList(pages)

	page := NewListPage(dir, title, pages)
	page.View = view
	return page, nil
}
