package powerlinx_test

import (
	"fmt"
	"testing"

	"github.com/stewartad/powerlinx"
)

func checkPageCount(expected, actual int) error {
	if actual != expected {
		return fmt.Errorf("Created %d pages, expected %d\n", actual, expected)
	}
	return nil
}

func checkTemplate(url, expected, actual string) error {
	if actual != expected {
		return fmt.Errorf("%s: incorrect template %s, expected %s\n", url, actual, expected)
	}
	return nil
}

// Check generation of aggregate pages

func TestSimpleGenerator(t *testing.T) {
	urls := []string{"/about", "/now"}
	site, pageCount := createSite(urls, []string{})
	err := site.Build()
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Log(site)
	err = checkPageCount(pageCount, len(site.Pages))
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestAggregatePageSimple(t *testing.T) {
	urls, listUrls := createDir("notes", 0, 2)
	site, pageCount := createSite(urls, listUrls)
	err := site.Build()
	t.Log(site)
	if err != nil {
		t.Fatalf(err.Error())
	}
	err = checkPageCount(pageCount, len(site.Pages))
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(site.Pages["/notes/index"].Metadata.Links) > 2 {
		t.Fatalf("Linked to %d pages, expected %d\n", len(site.Pages["/notes"].Metadata.Links), 2)
	}
}

func TestAggregatePageSimple2(t *testing.T) {
	depth := 2
	links := 4

	listUrls, urls := createDir("notes", depth, links)
	site, pageCount := createSite(urls, listUrls)
	err := site.Build()
	t.Log(site, pageCount, urls, listUrls)
	if err != nil {
		t.Fatalf(err.Error())
	}
	err = checkPageCount(pageCount, len(site.Pages))
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(site.Pages["/notes/subdir1/index"].Metadata.Links) > links {
		t.Fatalf("Linked to %d pages, expected %d\n", len(site.Pages["/notes"].Metadata.Links), 2)
	}
}

func TestAggregatePageComplex(t *testing.T) {

}

func TestAggregatePageOverwrite(t *testing.T) {

}

func TestTemplateMatchingBase(t *testing.T) {
	urls, listUrls := createDir("notes", 0, 2)
	site, _ := createSite(urls, listUrls)
	err := site.Build()
	t.Log(site)
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedTemplates := map[string]string{
		"/index":          string(powerlinx.TMPL_INDEX),
		"/notes/index":    string(powerlinx.TMPL_LIST),
		"/notes/example0": string(powerlinx.TMPL_PAGE),
		"/notes/example1": string(powerlinx.TMPL_PAGE),
	}
	for url, tmpl := range expectedTemplates {
		err = checkTemplate(url, tmpl, site.Pages[url].SiteTmpl.Name)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func TestTemplateCustom(t *testing.T) {
	urls, listUrls := createDir("notes", 0, 2)
	site, _ := createSite(urls, listUrls)

	site.Pages["/notes/example1"].Metadata.TmplName = "example1"
	site.SiteTmpl["notes/example1"] = createTemplate("example1", "notes")

	err := site.Build()
	t.Log(site)
	if err != nil {
		t.Fatalf(err.Error())
	}

	expectedTemplates := map[string]string{
		"/index":          string(powerlinx.TMPL_INDEX),
		"/notes/index":    string(powerlinx.TMPL_LIST),
		"/notes/example0": string(powerlinx.TMPL_PAGE),
		"/notes/example1": "example1",
	}
	for url, tmpl := range expectedTemplates {
		err = checkTemplate(url, tmpl, site.Pages[url].SiteTmpl.Name)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func TestFeed(t *testing.T) {

}
