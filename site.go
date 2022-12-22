package powerlinx

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
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
	),
)

func NewSite(contentFs fs.FS, templateFs fs.FS, opts ...SiteOption) (Site, error) {
	templates, err := discoverTemplates(templateFs)
	if err != nil {
		log.Fatalln(err)
	}
	pageData, err := discoverPages(contentFs)
	if err != nil {
		log.Fatalln(err)
	}
	pageMap := make(map[string]*Page)
	for filePath, data := range pageData {

		// Generate metadata for aggregate pages that need to be generated
		if len(data) == 0 {
			log.Printf("Generating %s\n", filePath)
			if err != nil {
				log.Fatalln(err)
			}
			url, _ := FilepathToUrl(filePath)
			pageMap[url] = NewAggregatePage(url)
		} else {
			page, err := NewPageFromFile(filePath, data)
			if err != nil {
				log.Fatalln(err)
			}
			pageMap[page.Metadata.Url] = page
		}
	}
	c := NewConfig()
	for _, opt := range opts {
		opt.SetSiteOption(c)
	}
	site := Site{
		Config:   c,
		Pages:    pageMap,
		SiteTmpl: templates,
	}
	err = site.Build()

	return site, err
}

func discoverTemplates(templatesFs fs.FS) (map[string]*SiteTemplate, error) {
	templates := make(map[string]*SiteTemplate)
	err := fs.WalkDir(templatesFs, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalln(err)
		}
		// skip base templates
		if d.IsDir() || strings.Contains(filePath, "base") {
			return nil
		}

		tmplPath := path.Clean(filePath)
		fileName := path.Base(tmplPath)
		tmplType := strings.TrimPrefix(fileName, "_")
		tmplType = strings.TrimSuffix(tmplType, path.Ext(fileName))
		templates[tmplPath] = &SiteTemplate{
			Path:   tmplPath,
			Name:   path.Base(tmplPath),
			Type:   TemplateType(tmplType),
			Layout: baseLayout,
		}
		perr := templates[tmplPath].ParseTemplate(templatesFs)
		if perr != nil {
			return perr
		}

		log.Printf("Discovered Template %s", tmplPath)
		return nil
	})
	return templates, err
}

func discoverPages(contentFs fs.FS) (map[string][]byte, error) {
	pageMap := make(map[string][]byte)
	err := fs.WalkDir(contentFs, ".", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.Name() == "." {
			return nil
		}
		if d.IsDir() {
			pageMap[path.Join(filePath, "index")] = []byte{}
			return nil
		}
		pageMap[filePath], err = fs.ReadFile(contentFs, filePath)
		if err != nil {
			return err
		}
		return nil
	})
	return pageMap, err

}

// A Site holds all the information about this website
type Site struct {
	Pages    map[string]*Page
	Config   *SiteConfig
	SiteTmpl map[string]*SiteTemplate
}

// Build will discover templates, discover individual pages, then generate ListPages for each
// directory in s.contentFs that does not have an index.html or index.md file
func (s *Site) Build() error {
	if !s.Config.IncludeHidden {
		s.removeHiddenPages()
	}
	s.generateAggregatePages()
	err := s.applyTemplates()
	if err != nil {
		return err
	}
	return err
}

func (s *Site) removeHiddenPages() {
	for url, page := range s.Pages {
		if page.Metadata.Draft {
			delete(s.Pages, url)
		}
	}
}

func (s *Site) generateAggregatePages() {
	urls := s.getAllUrls()
	for url, page := range s.Pages {
		if page.Metadata.Generate {
			page.Metadata.Title = path.Base(path.Dir(url))
			links := getAllPagesInDir(path.Dir(url), urls)
			pages := []Page{}
			for _, x := range links {
				pages = append(pages, *s.Pages[x])
			}
			log.Printf("Generating Page %s, Links %v", page.Metadata.Url, links)
			sort.Sort(byTime(pages))
			page.Content = nil
		}
	}
}

func (s *Site) getAllUrls() []string {
	urls := []string{}
	for url := range s.Pages {
		urls = append(urls, url)
	}
	return urls
}

func (s *Site) applyTemplates() error {
	for _, page := range s.Pages {
		tmpl, err := s.getTmpl(page)
		log.Printf("Applying template %s to page %s\n", tmpl.Path, page.Metadata.Url)
		if err != nil {
			return err
		}
		page.SiteTmpl = tmpl
	}
	return nil
}

// starting with the deepest possible template location and moving up, search for existing templates
func (s *Site) getTmpl(p *Page) (*SiteTemplate, error) {
	tmplName := p.Metadata.TmplName
	tmplDir := path.Dir(strings.TrimPrefix(p.Metadata.Url, "/"))
	tmplPath := ""
	for tmplDir != "." && tmplDir != "/" {
		tmplPath = path.Join(tmplDir, tmplName)
		tmpl, exists := s.SiteTmpl[tmplPath]
		if exists {
			return tmpl, nil
		}
		tmplDir = path.Dir(tmplDir)
	}
	tmpl, exists := s.SiteTmpl[tmplName]
	if !exists {
		return nil, errors.New("Can't find base template " + tmplName)
	}
	return tmpl, nil
}

// GenerateSite writes the fully rendered HTML pages of the site to directory outdir
func (s *Site) GenerateSite(outdir string) error {
	err := recreateDir(outdir)
	if err != nil {
		log.Fatalln(err)
	}
	for url, page := range s.Pages {
		outPath := path.Join(outdir, url+".html")
		err = writePage(outPath, *page)
		if err != nil {
			return err
		}
	}
	return nil
}

// deletes the target directory then creates an empty one in its place
func recreateDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.Mkdir(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// func (s *Site) GenerateFeed(pageDir string, outdir string, f *feeds.Feed) error {

// 	pages := getAllPagesInDir(pageDir, s.Pages)
// 	f.Items = []*feeds.Item{}
// 	for _, p := range pages {
// 		if path.Base(p.Metadata.Url) != "index" {
// 			item := &feeds.Item{
// 				Title:   p.Metadata.Title,
// 				Created: p.Metadata.CreatedAt,
// 				Link:    &feeds.Link{Href: "http://" + path.Join(s.Config.BaseUrl, p.Metadata.Url)},
// 			}
// 			f.Items = append(f.Items, item)
// 		}
// 	}
// 	atom, err := f.ToAtom()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	outFile, err := createFile(path.Join(outdir, "feed.xml"))
// 	if err != nil {
// 		return err
// 	}
// 	defer outFile.Close()
// 	fileWriter := bufio.NewWriter(outFile)
// 	_, err = fileWriter.WriteString(atom)
// 	if err != nil {
// 		return err
// 	}
// 	err = fileWriter.Flush()
// 	if err != nil {
// 		return err
// 	}
// 	return nil

// }
