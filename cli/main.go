package main

import (
	"log"
	"net/http"
	"os"

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

func main() {
	content := os.DirFS("data")
	templates := os.DirFS("templates")
	assets := os.DirFS("assets")
	// site := powerlinx.NewSite(content, templates, powerlinx.IncludeDrafts())
	site := powerlinx.NewSite(content, templates)

	// TODO: add views
	// need a way to automatically detect which pages have special templates
	// maybe apply templates of the same name as the directory?
	//

	site.Build()

	err := site.GenerateSite("pub")
	if err != nil {
		panic(err)
	}
	log.Println("Generated site in ./pub")

	fileserver := http.FileServer(HTMLDir{http.Dir("pub/")})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	http.Handle("/", http.StripPrefix("/", fileserver))

	log.Fatal((http.ListenAndServe(":8080", nil)))
}
