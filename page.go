package powerlinx

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
	"time"
)

type TemplateType string

const TMPL_PAGE TemplateType = "_single.html"
const TMPL_LIST TemplateType = "_list.html"
const TMPL_INDEX TemplateType = "_index.html"

func (t TemplateType) FileName() string {
	return fmt.Sprintf("_%s.html", t)
}

type templateLayout string

const baseLayout templateLayout = "layout.html"

// NewSiteTemplate creates a site template based on
// determines Type based on the filename.
//
// templates/_index.html for index at /
// templates/_single.html for individual pages
// templates/x/_list.html for index of directory x
// templates/x/_single.html for individual pages in directory x
// TODO: y.html for y.md (one-off templates but include base)
// This pattern continues for any number of directories
type SiteTemplate struct {
	Name     string
	Type     TemplateType
	Path     string
	Layout   templateLayout
	Template *template.Template
}

func (t *SiteTemplate) ParseTemplate(fs fs.FS) error {
	tmpl, err := template.ParseFS(fs, t.Path, "base/*.html")
	if err != nil {
		return err
	}
	t.Template = tmpl
	return nil
}

type PageMetadata struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	ContentID string    `json:"type"`
	Draft     bool      `json:"draft"`
	TmplName  string    `json:"template"`
	Url       string
	generate  bool
	// TODO: yaml metadata
}

type Page struct {
	Metadata PageMetadata
	Content  interface{}
	SiteTmpl *SiteTemplate
}

func (p *Page) Render(w io.Writer) error {
	return p.SiteTmpl.Template.ExecuteTemplate(w, string(p.SiteTmpl.Layout), *p)
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
	return t[j].Metadata.CreatedAt.Before(t[i].Metadata.CreatedAt)
}

func DefaultTemplateName(url string) string {
	if path.Base(url) == "index" {
		return string(TMPL_INDEX)
	} else {
		return string(TMPL_PAGE)
	}
}

func NewPageFromFile(filePath string, data []byte) (*Page, error) {
	// Split into metadata and body
	arr := bytes.Split(data, []byte("---"))
	var metadata PageMetadata
	var err error
	var body []byte
	if len(arr) >= 3 {
		// Parse metadata
		meta := arr[1]
		metadata, err = parseMetadata(meta)
		if err != nil {
			return nil, err
		}
		body = arr[2]
	} else {
		body = arr[0]
	}

	url, fileType := FilepathToUrl(filePath)
	metadata.Url = url
	if metadata.TmplName == "" {
		metadata.TmplName = DefaultTemplateName(metadata.Url)
	}
	// Convert content to HTML
	var content interface{}
	if fileType == ".md" {
		content, err = convertMdToHTML(body)
		if err != nil {
			return nil, err
		}
	} else if fileType == ".html" {
		content = template.HTML(string(body))
	} else {
		return nil, err
	}
	return &Page{
		Metadata: metadata,
		Content:  content,
	}, err
}

func NewAggregatePage(url string) *Page {
	return &Page{
		Metadata: PageMetadata{
			Title:     path.Base(url),
			CreatedAt: time.Now(),
			Url:       url,
			TmplName:  string(TMPL_LIST),
			generate:  true,
		},
	}
}
