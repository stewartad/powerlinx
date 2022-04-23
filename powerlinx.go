package powerlinx

import (
	"bufio"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Site struct {
	content     fs.FS
	templates   fs.FS
	assets      fs.FS
	PageMap     map[string]*Page
	Views       map[string]*View
	RecentPosts []*Page
}

func NewSite(content, templates, assets fs.FS) *Site {
	site := Site{
		content:   content,
		templates: templates,
		assets:    assets,
		PageMap:   make(map[string]*Page),
		Views:     make(map[string]*View),
	}
	err := site.loadAllStaticPages()
	if err != nil {
		log.Fatal(err)
	}
	site.RecentPosts = site.getRecentBlogPosts(10)
	return &site
}

func (s *Site) AddView(name string, v *View) {
	s.Views[name] = v
}

type Page struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"date"`
	Type      string    `json:"type"`
	Body      template.HTML
	Url       string
}

func (s *Site) loadAllStaticPages() error {
	err := fs.WalkDir(s.content, "data", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		} else {
			page, ferr := s.parseSinglePage(path)
			if ferr != nil {
				return ferr
			}
			s.PageMap[strings.TrimPrefix(path, "data/")] = page
		}
		return nil
	})
	return err
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

func (s *Site) getRecentBlogPosts(count int) []*Page {
	all := make([]*Page, 0, len(s.PageMap))
	for _, value := range s.PageMap {
		if value.Type == "post" {
			all = append(all, value)
		}
	}
	sort.Sort(byTime(all))
	if count > len(s.PageMap) {
		return all
	}
	return all[:count]
}

func (s *Site) parseSinglePage(path string) (*Page, error) {
	file, err := s.content.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	metadata, body := parseMetadata(file)
	bodyHTML := template.HTML(string(body))
	url := strings.TrimPrefix(strings.TrimSuffix(path, ".html"), "data")
	page := &Page{Body: bodyHTML, Url: url}
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
		}
	}
	return metadata, body
}

type View struct {
	Template *template.Template
	Layout   string
}

func (v *View) Render(w http.ResponseWriter, data interface{}) error {
	return v.Template.ExecuteTemplate(w, v.Layout, data)
}

func (s *Site) NewView(layout string, lastTmpl string) *View {
	t, err := template.ParseFS(s.templates, lastTmpl, "templates/base/*.html")
	if err != nil {
		log.Fatal(err)
	}

	return &View{Template: t, Layout: layout}
}
