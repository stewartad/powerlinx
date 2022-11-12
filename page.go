package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const TMPL_PAGE = "single"
const TMPL_LIST = "list"
const TMPL_INDEX = "index"

/*
 * Util functions relating to the creation and parsing of individual pages
 */

type Page interface {
	Render(w io.Writer) error
	// TODO: pass view to render
}

// A DetailPage contains metadata and content for a single webpages
// Metadata is standard json surrounded by "---"
type DetailPage struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	Type      string    `json:"type"`
	Draft     bool      `json:"draft"`
	Url       string
	Body      interface{}
	View      *View
}

func (p *DetailPage) Render(w io.Writer) error {
	return p.View.Render(w, p)
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
	Title string
	Url   string
	Pages []*DetailPage
	View  *View
}

func (p *ListPage) Render(w io.Writer) error {
	return p.View.Render(w, p)
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
