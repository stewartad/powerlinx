package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"time"
)

/*
 * Util functions relating to the creation and parsing of individual pages
 */

// A Page contains metadata and content for a single webpages
// Metadata is standard json surrounded by "---"
type Page struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	Type      string    `json:"type"`
	Url       string
	Body      interface{}
	View      *View
}

type ListPage struct {
	Title string
	Url   string
	Pages []*Page
	View  *View
}

// separateMetadata separates JSON metadata from page content.
// Metadata is at the top of the file, surrounded by "---"
func separateMetadata(r io.Reader) ([]byte, []byte) {
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

func parseMetadata(data []byte) (*Page, error) {
	page := Page{}
	if len(data) > 0 {
		err := json.Unmarshal(data, &page)
		if err != nil {
			return nil, err
		}
	}
	return &page, nil
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
