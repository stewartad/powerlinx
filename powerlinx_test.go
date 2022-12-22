package powerlinx_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/stewartad/powerlinx"
)

func createSite(dir string, opts ...powerlinx.SiteOption) powerlinx.Site {
	content := os.DirFS(path.Join(dir, "content"))
	templates := os.DirFS(path.Join(dir, "templates"))

	// site := powerlinx.NewSite(content, templates, powerlinx.IncludeDrafts())
	site, err := powerlinx.NewSite(content, templates, opts...)
	if err != nil {
		panic(err)
	}
	return site
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

// Check generation of aggregate pages

func TestSimpleGenerator(t *testing.T) {
	urls := []string{"/index"}
	pages := map[string]*powerlinx.Page{}
	for _, u := range urls {
		pages[u] = createPage("index", "/index", "_index.html")
	}

	templates := map[string]*powerlinx.SiteTemplate{
		string(powerlinx.TMPL_INDEX): createTemplate(powerlinx.TMPL_INDEX, ""),
	}
	site := powerlinx.Site{
		Pages:    pages,
		SiteTmpl: templates,
		Config:   powerlinx.NewConfig(),
	}
	err := site.Build()
	if err != nil {
		t.Fatalf("Error building site %s\n", err)
	}
}
