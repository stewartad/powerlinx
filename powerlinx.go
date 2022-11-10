package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
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
	err := site.GenerateViews()
	if err != nil {
		log.Fatal(err)
	}
	err = site.loadAllStaticPages()
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

func (s *Site) loadAllStaticPages() error {
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
	page.View = s.getTemplate(path.Dir(page.Url), "page")

	log.Printf("Loading Page %s, Url %s", filePath, page.Url)

	return page, nil
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

func (s *Site) DiscoverTemplates() error {
	// walk template dir
	err := fs.WalkDir(s.templates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalln(err)
		}
		// skip base templates
		if strings.Contains(path, "base") || path == "." {
			return nil
		}
		if d.IsDir() {
			s.StaticViews[path] = s.NewView("layout.html", path+"/_index.html")
		} else {
			s.StaticViews[path] = s.NewView("layout.html", path)
		}
		return nil
	})
	return err
}

func (s *Site) getTemplate(pageDir string, templateName string) *View {

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

// looks for _index.html and _single.html
//
func (s *Site) GenerateViews() error {
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

func createHTMLFile(outPath string) (*os.File, error) {
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

func (s *Site) GenerateSite() {
	err := os.Mkdir("pub", 0755)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
	}

	for url, page := range s.PageMap {
		outPath := path.Join("pub" + url + ".html")
		outFile, err := createHTMLFile(outPath)
		if err != nil {
			// TODO: better handling
			panic(err)
		}

		fileWriter := bufio.NewWriter(outFile)
		// TODO: determine the real template to render
		// TODO: properly generate blog page

		err = page.View.Render(fileWriter, page)
		if err != nil {
			panic(err)
		}
		err = fileWriter.Flush()
		if err != nil {
			panic(err)
		}
	}
}
