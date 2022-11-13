package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"strings"
	"time"
)

/*
 * Util functions relating to the creation and parsing of individual pages
 */

type Page interface {
	Render(w io.Writer) error
	Type() templateType
	Path() string // rename to Url()
	Hidden() bool
	Content() interface{}
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
	View        *View
	Template    *SiteTemplate
}

func (p *DetailPage) Render(w io.Writer) error {
	return p.Template.Template.ExecuteTemplate(w, string(p.Template.Layout), p.Content())
}

func (p *DetailPage) Type() templateType {
	if path.Base(p.Url) == "index" {
		return TMPL_INDEX
	}
	return TMPL_PAGE
}

func (p *DetailPage) Path() string {
	return p.Url
}

func (p *DetailPage) Hidden() bool {
	return p.Draft
}

func (p *DetailPage) Content() interface{} {
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

func NewDetailPage(file fs.File, path string) (*DetailPage, error) {
	metadata, body := separateMetadata(file)
	filetype := filepath.Ext(path)
	page := DetailPage{}
	if len(metadata) > 0 {
		err := json.Unmarshal(metadata, &page)
		if err != nil {
			return nil, err
		}
	}
	page.Body = convertToHTML(body, filetype)
	page.Url = strings.TrimSuffix("/"+path, filetype)
	return &page, nil
}

type ListPage struct {
	Title    string
	Url      string
	Pages    []*DetailPage
	View     *View
	Template *SiteTemplate
}

func (p *ListPage) Render(w io.Writer) error {
	return p.Template.Template.ExecuteTemplate(w, string(p.Template.Layout), p.Content())
}

func (p *ListPage) Type() templateType {
	return TMPL_LIST
}

func (p *ListPage) Path() string {
	return p.Url
}

func (p *ListPage) GetTemplatePath() string {
	tmplName := p.Type().FileName()
	dir := path.Dir(p.Path())
	if dir == "." {
		return tmplName
	}
	return path.Join(dir, tmplName)
}

func (p *ListPage) Hidden() bool {
	return false
}

func (p *ListPage) Content() interface{} {
	return struct {
		Title string
		Url   string
		Pages []*DetailPage
	}{
		Title: p.Title,
		Url:   p.Url,
		Pages: p.Pages,
	}
}

func NewListPage(dir string, title string, pages []*DetailPage) *ListPage {
	// turn title to title case
	title = strings.ToUpper(string(title[0])) + string(title[1:])
	return &ListPage{
		Title: title,
		Url:   "/" + dir,
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
