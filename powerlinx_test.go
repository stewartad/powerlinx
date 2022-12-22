package powerlinx_test

import (
	"path"
	"testing"
	"time"

	"github.com/stewartad/powerlinx"
)

func createDefaultSite() powerlinx.Site {
	return powerlinx.Site{
		Pages: map[string]*powerlinx.Page{
			"/index": createPage("index", "/index", "_index.html"),
		},
		SiteTmpl: defaultTemplates(),
		Config:   powerlinx.NewConfig(),
	}
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

func createSinglePages(urls []string, pages map[string]*powerlinx.Page) {
	for _, u := range urls {
		pages[u] = createPage(path.Base(u), u, string(powerlinx.TMPL_PAGE))
	}
}

func createAggregatePages(urls []string, pages map[string]*powerlinx.Page) {
	for _, u := range urls {
		page := createPage(path.Base(u), u, string(powerlinx.TMPL_PAGE))
		page.Metadata.Generate = true
		pages[u] = page
	}
}

func defaultTemplates() map[string]*powerlinx.SiteTemplate {
	return map[string]*powerlinx.SiteTemplate{
		string(powerlinx.TMPL_INDEX): createTemplate(powerlinx.TMPL_INDEX, ""),
		string(powerlinx.TMPL_LIST):  createTemplate(powerlinx.TMPL_LIST, ""),
		string(powerlinx.TMPL_PAGE):  createTemplate(powerlinx.TMPL_PAGE, ""),
	}
}

// Check generation of aggregate pages

func TestSimpleGenerator(t *testing.T) {
	urls := []string{"/about", "/now"}

	site := createDefaultSite()
	createSinglePages(urls, site.Pages)
	err := site.Build()
	if err != nil {
		t.Fatalf("Error building site %s\n", err)
	}
	t.Log(site)
	if len(site.Pages) > len(urls)+1 {
		t.Fatalf("Created %d pages, expected %d\n", len(site.Pages), len(urls))
	}
}

func TestAggregatePageSimple(t *testing.T) {
	urls := []string{"/notes/example1", "notes/example2"}
	listUrls := []string{"/notes"}
	pageCount := len(urls) + len(listUrls) + 1
	site := createDefaultSite()
	createSinglePages(urls, site.Pages)
	createAggregatePages(listUrls, site.Pages)
	err := site.Build()
	if err != nil {
		t.Fatalf("Error building site %s\n", err)
	}
	t.Log(site)
	if len(site.Pages) > pageCount {
		t.Fatalf("Created %d pages, expected %d\n", len(site.Pages), len(urls))
	}
	if len(site.Pages["/notes"].Metadata.Links) > 2 {
		t.Fatalf("Linked to %d pages, expected %d\n", len(site.Pages["/notes"].Metadata.Links), 2)
	}
}

func TestAggregatePageComplex(t *testing.T) {

}

func TestAggregatePageOverwrite(t *testing.T) {

}

func TestTemplateMatchingSimple(t *testing.T) {

}

func TestTemplateMatchingComplex(t *testing.T) {

}

func TestFeed(t *testing.T) {

}
