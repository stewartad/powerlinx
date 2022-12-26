package powerlinx_test

import (
	"fmt"
	"path"
	"time"

	"github.com/gorilla/feeds"
	"github.com/stewartad/powerlinx"
)

func createSite(urls, listUrls []string) (powerlinx.Site, int) {
	pages := map[string]*powerlinx.Page{
		"/index": createPage("index", "/index", "_index.html"),
	}
	for _, u := range urls {
		pages[u] = createPage(path.Base(u), u, string(powerlinx.TMPL_PAGE))
	}
	for _, u := range listUrls {
		page := createPage(path.Base(u), u, string(powerlinx.TMPL_LIST))
		page.Metadata.Generate = true
		pages[u] = page
	}
	return powerlinx.Site{
		Pages:    pages,
		SiteTmpl: defaultTemplates(),
		Config:   powerlinx.NewConfig(),
		Feeds:    make(map[string]*feeds.Feed),
	}, len(urls) + len(listUrls) + 1
}

// createDir returns all file paths in the dir, then all the subdirs
func createDir(name string, depth, links int) ([]string, []string) {
	urls := []string{}
	listUrls := []string{}
	dir := "/" + name
	subdir := dir
	for i := 0; i <= depth; i++ {
		if i > 0 {
			subdir = path.Join(dir, fmt.Sprintf("subdir%d", i))
		}
		listUrls = append(listUrls, path.Join(subdir, "index"))
		for j := 0; j < links; j++ {
			urls = append(urls, path.Join(subdir, fmt.Sprintf("example%d", j)))
		}
	}
	return urls, listUrls
}

// Create dummy pages
func createPage(title, url, tmplName string) *powerlinx.Page {
	return &powerlinx.Page{
		Metadata: powerlinx.PageMetadata{
			Title:     title,
			CreatedAt: time.Now(),
			Url:       url,
			TmplName:  tmplName,
		},
		Content: "hello",
	}
}

// Create dummy templates
func createTemplate(tmplName powerlinx.TemplateType, dir string) *powerlinx.SiteTemplate {
	return &powerlinx.SiteTemplate{
		Name: string(tmplName),
		Type: tmplName,
		Path: path.Join(dir, string(tmplName)),
	}
}

func defaultTemplates() map[string]*powerlinx.SiteTemplate {
	return map[string]*powerlinx.SiteTemplate{
		string(powerlinx.TMPL_INDEX): createTemplate(powerlinx.TMPL_INDEX, ""),
		string(powerlinx.TMPL_LIST):  createTemplate(powerlinx.TMPL_LIST, ""),
		string(powerlinx.TMPL_PAGE):  createTemplate(powerlinx.TMPL_PAGE, ""),
	}
}
