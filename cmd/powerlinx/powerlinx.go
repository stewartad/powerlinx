package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/stewartad/powerlinx"
)

type HTMLDir struct {
	d http.Dir
}

func (d HTMLDir) Open(name string) (http.File, error) {
	f, err := d.d.Open(name + ".html")
	if os.IsNotExist(err) {
		if f, err := d.d.Open(name); err == nil {
			return f, nil
		}
	}
	return f, err
}

func createSite(dir string) (powerlinx.Site, error) {
	dir = path.Clean(dir)
	content := os.DirFS(path.Join(dir, "content"))
	templates := os.DirFS(path.Join(dir, "templates"))
	// assets := os.DirFS(path.Join(dir, "assets"))
	return powerlinx.NewSite(content, templates)
}

func startServer(dir string) {
	assets := os.DirFS(path.Join(dir, "assets")) // TODO: copy assets to public
	public := path.Join(dir, "public")
	fileserver := http.FileServer(HTMLDir{http.Dir(public)})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	http.Handle("/", http.StripPrefix("/", fileserver))

	log.Fatal((http.ListenAndServe(":8080", nil)))
}

func main() {
	var sitedir string
	var server bool
	flag.StringVar(&sitedir, "f", ".", "directory to read site data from")
	flag.BoolVar(&server, "s", false, "start server")
	flag.Parse()

	pubdir := path.Join(sitedir, "public")
	site, err := createSite(sitedir)
	if err != nil {
		log.Fatalf("error building site: %s", err.Error())
	}

	log.Printf("Generating site in %s\n", pubdir)
	err = site.GenerateSite(pubdir)
	if err != nil {
		log.Fatalf("error writing site to %s: %s\n", pubdir, err.Error())
	}

	if server {
		startServer(sitedir)
	}
}
