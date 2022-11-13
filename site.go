package powerlinx

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path"
	"strings"
	"text/template"

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
	contentFs     fs.FS
	templatesFs   fs.FS
	PageMap       map[string]Page
	SortedPages   []Page
	Config        *SiteConfig
	SiteTemplates map[string]*SiteTemplate
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
		contentFs:     content,
		templatesFs:   templates,
		PageMap:       map[string]Page{},
		SiteTemplates: map[string]*SiteTemplate{},
		Config:        c,
	}
}

// Build will discover templates, create views for the templates, then discover content pages to be parsed, then generate aggregate pages
func (s *Site) Build() {
	log.Println("Discovering Templates")
	err := s.discoverTemplates()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Discovering Pages")
	err = s.discoverPages()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Discovering ListPages")
	err = s.discoverListPages()
	if err != nil {
		log.Fatal(err)
	}
	s.SortedPages = s.sortAllPages()
}

// GetRecentPages returns a slice of Pages of the specified pageType,
// as defined in a Page's metadata.
// If pageType is "", Pages of any type are returned
// The length of the returned slice is either count or the length of s.PageMap,
// whichever is smaller
func (s *Site) GetRecentPages(count int, pageType string) []Page {
	all := make([]Page, 0, count)

	for _, page := range s.SortedPages {
		if getContentType(page) == pageType || pageType == "" {
			all = append(all, page)
		}
		if len(all) == count {
			break
		}
	}
	return all
}

type templateType string

const TMPL_PAGE templateType = "single"
const TMPL_LIST templateType = "list"
const TMPL_INDEX templateType = "index"

func (t templateType) FileName() string {
	return fmt.Sprintf("_%s.html", t)
}

type templateLayout string

const baseLayout templateLayout = "layout.html"

type SiteTemplate struct {
	Name     string
	Type     templateType
	Path     string
	Layout   templateLayout
	Template *template.Template
}

// NewSiteTemplate creates a site template based on
// determines Type based on the filename.
//
// templates/_index.html for index at /
// templates/_single.html for individual pages
// templates/x/_list.html for index of directory x
// templates/x/_single.html for individual pages in directory x
// TODO: y.html for y.md (one-off templates but include base)
// This pattern continues for any number of directories
func NewSiteTemplate(filePath string) *SiteTemplate {
	cleanPath := path.Clean(filePath)
	fileName := path.Base(cleanPath)
	tmplType := strings.TrimPrefix(fileName, "_")
	tmplType = strings.TrimSuffix(tmplType, path.Ext(fileName))

	return &SiteTemplate{
		Path: cleanPath,
		Name: path.Base(cleanPath),
		Type: templateType(tmplType),
	}
}

func (t *SiteTemplate) ParseTemplate(fs fs.FS) error {
	tmpl, err := template.ParseFS(fs, t.Path, "base/*.html")
	if err != nil {
		return err
	}
	t.Template = tmpl
	return nil
}

func (s *Site) discoverTemplates() error {
	err := fs.WalkDir(s.templatesFs, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalln(err)
		}
		// skip base templates
		if d.IsDir() || strings.Contains(filePath, "base") {
			return nil
		}

		tmpl := NewSiteTemplate(filePath)
		tmpl.Layout = baseLayout
		perr := tmpl.ParseTemplate(s.templatesFs)
		if err != nil {
			return perr
		}
		s.SiteTemplates[tmpl.Path] = tmpl
		log.Printf("Discovered Template %s", tmpl.Path)
		return nil
	})
	return err
}

func (s *Site) discoverPages() error {
	err := fs.WalkDir(s.contentFs, ".", func(path string, d fs.DirEntry, err error) error {
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
		log.Printf("Discovered Page %s, Url %s, Template %s", path, getUrl(page), page.getTemplate().Path)
		if !isHidden(page) || s.Config.IncludeHidden {
			s.PageMap[getUrl(page)] = page
		}

		return nil
	})

	return err
}

func (s *Site) discoverListPages() error {
	err := fs.WalkDir(s.contentFs, ".", func(path string, d fs.DirEntry, err error) error {
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
		// only generate if page doesn't already exist
		_, exists := s.PageMap[getUrl(page)]
		if exists {
			return nil
		}
		s.PageMap[getUrl(page)] = page
		return nil
	})
	return err
}

func (s *Site) createPageFromFile(filePath string) (Page, error) {
	file, err := s.contentFs.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	page, err := NewDetailPage(file, filePath)
	if err != nil {
		return nil, err
	}
	page.Template, err = s.getTemplate(page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// starting with the deepest possible template location and moving up, search for existing templates
func (s *Site) getTemplate(p Page) (*SiteTemplate, error) {
	tmplName := tmplType(p).FileName()
	tmplDir := path.Dir(getUrl(p))
	for tmplDir != "." && tmplDir != "/" {
		tmplPath := path.Join(tmplDir, tmplName)
		template, exists := s.SiteTemplates[tmplPath]
		if exists {
			return template, nil
		}
		tmplDir = path.Join(path.Dir(tmplDir))
	}
	template, exists := s.SiteTemplates[tmplName]
	if !exists {
		return nil, errors.New("Can't find base template " + tmplName)
	}
	return template, nil
}

func (s *Site) getAllPagesInDir(dir string) []Page {
	// TODO: check whether to include subdirs or not
	pages := make([]Page, 0)
	for url, page := range s.PageMap {
		if strings.HasPrefix(url, dir) {
			pages = append(pages, page)
		}
	}
	return pages
}

func (s *Site) createListPage(dir string, title string) (Page, error) {
	pages := s.getAllPagesInDir(path.Clean("/" + dir))
	sortPageList(pages)

	page := NewListPage(dir, title, pages)
	template, err := s.getTemplate(page)
	if err != nil {
		return nil, err
	}
	page.Template = template
	return page, nil
}
