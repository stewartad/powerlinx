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
	templateType() templateType
	path() string // rename to Url()
	hidden() bool
	content() interface{}
	date() time.Time
	getContentType() string
	getTemplate() *SiteTemplate
}

func tmplType(p Page) templateType { return p.templateType() }
func getUrl(p Page) string         { return p.path() }
func isHidden(p Page) bool         { return p.hidden() }
func getDate(p Page) time.Time     { return p.date() }
func getContentType(p Page) string { return p.getContentType() }

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

func (p ListPage) content() interface{} {
	return struct {
		Title string
		Url   string
		Pages []Page
	}{
		Title: p.Title,
		Url:   p.Url,
		Pages: p.Pages,
	}
}

func NewListPage(dir string, title string, pages []Page) ListPage {
	// turn title to title case
	title = strings.ToUpper(string(title[0])) + string(title[1:])
	return ListPage{
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
