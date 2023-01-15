package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/stewartad/powerlinx"
	"gopkg.in/yaml.v2"
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

func readConfig(configpath string) (*powerlinx.SiteConfig, error) {
	file, err := os.ReadFile(configpath)
	if err != nil {
		return nil, err
	}

	cfg := powerlinx.SiteConfig{}

	yaml.UnmarshalStrict(file, &cfg)
	log.Println(cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, err
}

func createSite(dir string, configfile string) (*powerlinx.Site, error) {
	dir = path.Clean(dir)
	content := os.DirFS(path.Join(dir, "content"))
	templates := os.DirFS(path.Join(dir, "templates"))
	// assets := os.DirFS(path.Join(dir, "assets"))
	cfg, err := readConfig(configfile)
	if err != nil {
		return nil, err
	}
	site, err := powerlinx.NewSite(content, templates)
	site.Config = cfg
	return &site, err

}

func startServer(dir string, port string) {
	assets := os.DirFS(path.Join(dir, "assets")) // TODO: copy assets to public
	public := path.Join(dir, "public")
	fileserver := http.FileServer(HTMLDir{http.Dir(public)})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	http.Handle("/", http.StripPrefix("/", fileserver))

	log.Fatal((http.ListenAndServe(":"+port, nil)))
}

func main() {
	var sitedir string
	var server bool
	var config string
	var port string

	flag.StringVar(&sitedir, "f", ".", "directory to read site data from")
	flag.BoolVar(&server, "s", false, "start server")
	flag.StringVar(&config, "c", "", "path to configuration file")
	flag.StringVar(&port, "p", "8080", "port to start server on")
	flag.Parse()

	if config == "" {
		config = path.Join(sitedir, "config.yml")
	}

	pubdir := path.Join(sitedir, "public")
	site, err := createSite(sitedir, config)
	if err != nil {
		log.Fatalf("error parsing site: %s", err.Error())
	}

	err = site.Build()
	if err != nil {
		log.Fatalf("error building site: %s", err.Error())
	}

	log.Printf("Generating site in %s\n", pubdir)
	err = site.GenerateSite(pubdir)
	if err != nil {
		log.Fatalf("error writing site to %s: %s\n", pubdir, err.Error())
	}

	if server {
		startServer(sitedir, port)
	}
}
