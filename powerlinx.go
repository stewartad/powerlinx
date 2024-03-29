package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
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
	),
)

type OptionName string

type SiteConfig struct {
	IncludeHidden bool
	BaseUrl       string
}

func NewConfig() *SiteConfig {
	return &SiteConfig{
		IncludeHidden: false,
		BaseUrl:       "localhost:8080",
	}
}

type SiteOption interface {
	SetSiteOption(*SiteConfig)
}

type includeDrafts struct{}

func (o *includeDrafts) SetSiteOption(c *SiteConfig) {
	c.IncludeHidden = true
}

func IncludeDrafts() interface {
	SiteOption
} {
	return &includeDrafts{}
}

type setBaseUrl struct {
	url string
}

func (o *setBaseUrl) SetSiteOption(c *SiteConfig) {
	c.BaseUrl = o.url
}

func SetBaseUrl(url string) interface {
	SiteOption
} {
	return &setBaseUrl{
		url: url,
	}
}

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
// its root directory should contain layouts for individual pages
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

// Build will discover templates, discover individual pages, then generate ListPages for each
// directory in s.contentFs that does not have an index.html or index.md file
func (s *Site) Build() error {
	log.Println("Discovering Templates")
	err := s.discoverTemplates()
	if err != nil {
		return err
	}
	log.Println("Discovering Pages")
	err = s.discoverPages()
	if err != nil {
		return err
	}

	log.Println("Discovering ListPages")
	err = s.generateListPages()
	if err != nil {
		return err
	}
	s.SortedPages = s.sortAllPages()
	return nil
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

type Page interface {
	Render(w io.Writer) error
	templateType() templateType
	path() string
	hidden() bool
	content() interface{}
	date() time.Time
	getContentType() string
	getTemplate() *SiteTemplate
	title() string
}

func tmplType(p Page) templateType     { return p.templateType() }
func getUrl(p Page) string             { return p.path() }
func isHidden(p Page) bool             { return p.hidden() }
func getDate(p Page) time.Time         { return p.date() }
func getContentType(p Page) string     { return p.getContentType() }
func getTemplate(p Page) *SiteTemplate { return p.getTemplate() }
func getTitle(p Page) string           { return p.title() }

func (s *Site) sortAllPages() []Page {
	all := make([]Page, 0, len(s.PageMap))
	for _, value := range s.PageMap {
		all = append(all, value)
	}
	sort.Sort(byTime(all))
	return all
}

func sortPageList(pages []Page) []Page {
	sort.Sort(byTime(pages))
	return pages
}

// Create Sort Interface for Pages
type byTime []Page

func (t byTime) Len() int {
	return len(t)
}

func (t byTime) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t byTime) Less(i, j int) bool {
	return getDate(t[j]).Before(getDate(t[i]))
}

// A DetailPage contains metadata and content for a single webpages
// Metadata is standard json surrounded by "---"
type DetailPage struct {
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"date"`
	ContentType string    `json:"type"`
	Draft       bool      `json:"draft"`
	Url         string
	Body        interface{}
	Template    *SiteTemplate
}

func (p DetailPage) Render(w io.Writer) error {
	return p.Template.Template.ExecuteTemplate(w, string(p.Template.Layout), p.content())
}

func (p DetailPage) templateType() templateType {
	if path.Base(p.Url) == "index" {
		return TMPL_INDEX
	}
	return TMPL_PAGE
}

func (p DetailPage) path() string { return p.Url }

func (p DetailPage) hidden() bool { return p.Draft }

func (p DetailPage) date() time.Time { return p.CreatedAt }

func (p DetailPage) getContentType() string { return p.ContentType }

func (p DetailPage) getTemplate() *SiteTemplate { return p.Template }

func (p DetailPage) title() string { return p.Title }

func (p DetailPage) content() interface{} {
	return struct {
		Title     string
		CreatedAt time.Time
		Url       string
		Body      interface{}
	}{
		Title:     p.Title,
		CreatedAt: p.CreatedAt,
		Url:       p.Url,
		Body:      p.Body,
	}
}

func NewDetailPage(file fs.File, path string) (DetailPage, error) {
	metadata, body := separateMetadata(file)
	filetype := filepath.Ext(path)
	page := DetailPage{}
	if len(metadata) > 0 {
		err := json.Unmarshal(metadata, &page)
		if err != nil {
			return DetailPage{}, err
		}
	}
	page.Body = convertToHTML(body, filetype)
	page.Url = strings.TrimSuffix("/"+path, filetype)
	return page, nil
}

type ListPage struct {
	Title    string
	Url      string
	Pages    []Page
	Template *SiteTemplate
}

func (p ListPage) Render(w io.Writer) error {
	return p.Template.Template.ExecuteTemplate(w, string(p.Template.Layout), p.content())
}

func (p ListPage) templateType() templateType { return TMPL_LIST }

func (p ListPage) path() string { return p.Url }

func (p ListPage) hidden() bool { return false }

func (p ListPage) date() time.Time { return time.Now() }

func (p ListPage) getContentType() string { return "list" }

func (p ListPage) getTemplate() *SiteTemplate { return p.Template }

func (p ListPage) title() string { return p.Title }

func (p ListPage) content() interface{} {
	pageContent := []interface{}{}
	for _, page := range p.Pages {
		pageContent = append(pageContent, page.content())
	}
	return struct {
		Title string
		Url   string
		Pages []interface{}
	}{
		Title: p.Title,
		Url:   p.Url,
		Pages: pageContent,
	}
}

func NewListPage(dir string, title string, pages []Page) ListPage {
	// turn title to title case
	title = strings.ToUpper(string(title[0])) + string(title[1:])
	return ListPage{
		Title: title,
		Url:   path.Join("/"+dir, "index"),
		Pages: pages,
	}
}

type metadata []byte
type body []byte

// separateMetadata separates JSON metadata from page content.
// Metadata is at the top of the file, surrounded by "---"
func separateMetadata(r io.Reader) (metadata, body) {
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

func convertToHTML(data []byte, filetype string) template.HTML {
	// if md, parse to html
	// if html, parse as-is
	if filetype == ".md" {
		var buf bytes.Buffer
		if err := markdown.Convert(data, &buf); err != nil {
			log.Panic(err)
		}
		return template.HTML(buf.String())

	} else if filetype == ".html" {
		return template.HTML(string(data))
	} else {
		log.Printf("Invalid filetype %s\n", filetype)
	}
	return ""
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
		if !isHidden(page) || s.Config.IncludeHidden {
			s.PageMap[getUrl(page)] = page
			log.Printf("Discovered Page %s, Url %s, Template %s", path, getUrl(page), getTemplate(page).Path)
		}

		return nil
	})

	return err
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

func (s *Site) generateListPages() error {
	err := fs.WalkDir(s.contentFs, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.Name() == "." || !d.IsDir() {
			return nil
		}

		pages := s.getAllPagesInDir("/" + filePath)
		sortPageList(pages)

		page := NewListPage(filePath, d.Name(), pages)
		template, ferr := s.getTemplate(page)
		if ferr != nil {
			return ferr
		}
		page.Template = template

		// only generate if page doesn't already exist
		currPage, exists := s.PageMap[getUrl(page)]
		if exists {
			log.Printf("Page already exists, skipped %s, using template %s", getUrl(currPage), getTemplate(currPage).Path)
			return nil
		}
		s.PageMap[getUrl(page)] = page
		log.Printf("Generated Page %s, Template %s", getUrl(page), page.Template.Path)
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
	tmplDir := strings.TrimPrefix(path.Dir(getUrl(p)), "/")
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

func createFile(outPath string) (*os.File, error) {
	err := os.MkdirAll(path.Dir(outPath), 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func writePage(outFile *os.File, page Page) error {
	fileWriter := bufio.NewWriter(outFile)
	err := page.Render(fileWriter)
	if err != nil {
		return err
	}
	err = fileWriter.Flush()
	if err != nil {
		return err
	}
	return nil
}

// GenerateSite writes the fully rendered HTML pages of the site to directory outdir
func (s *Site) GenerateSite(outdir string) error {
	err := os.RemoveAll(outdir)
	if err != nil {
		log.Println("could not delete pub")
	}

	err = os.Mkdir(outdir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
	}

	for url, page := range s.PageMap {
		outPath := path.Join(outdir + url + ".html")
		outFile, err := createFile(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		err = writePage(outFile, page)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Site) GenerateFeed(pageDir string, outdir string, f *feeds.Feed) error {
	pages := s.getAllPagesInDir(pageDir)
	f.Items = []*feeds.Item{}
	for _, p := range pages {
		if path.Base(getUrl(p)) != "index" {
			item := &feeds.Item{
				Title:   getTitle(p),
				Created: getDate(p),
				Link:    &feeds.Link{Href: "http://" + path.Join(s.Config.BaseUrl, getUrl(p))},
			}
			f.Items = append(f.Items, item)
		}
	}
	atom, err := f.ToAtom()
	if err != nil {
		log.Fatal(err)
	}
	outFile, err := createFile(path.Join(outdir, "feed.xml"))
	if err != nil {
		return err
	}
	defer outFile.Close()
	fileWriter := bufio.NewWriter(outFile)
	_, err = fileWriter.WriteString(atom)
	if err != nil {
		return err
	}
	err = fileWriter.Flush()
	if err != nil {
		return err
	}
	return nil

}
