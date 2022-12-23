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

func Yeq() {
	content := os.DirFS("testdata/yequari.com/content")
	templates := os.DirFS("testdata/yequari.com/templates")
	assets := os.DirFS("testdata/yequari.com/assets")
	// site := powerlinx.NewSite(content, templates, powerlinx.IncludeDrafts())
	site, err := powerlinx.NewSite(content, templates)
	if err != nil {
		panic(err)
	}

	log.Println("Generating Site in ./pub")
	err = site.GenerateSite("testdata/yequari.com/public")
	if err != nil {
		panic(err)
	}
	// log.Println("Generating Feed in ./pub")

	// now := time.Now()
	// feed := &feeds.Feed{
	// 	Title:       "yequari's blog",
	// 	Link:        &feeds.Link{Href: "http://" + site.Config.BaseUrl},
	// 	Description: "thoughts about anything and nothing",
	// 	Author:      &feeds.Author{Name: "yequari"},
	// 	Created:     now,
	// }
	// err = site.GenerateFeed("/blog", "pub", feed)
	// if err != nil {
	// 	panic(err)
	// }

	fileserver := http.FileServer(HTMLDir{http.Dir("testdata/yequari.com/public/")})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	http.Handle("/", http.StripPrefix("/", fileserver))

	log.Fatal((http.ListenAndServe(":8080", nil)))
}

func Basic() {
	content := os.DirFS("testdata/basic/content")
	templates := os.DirFS("testdata/basic/templates")
	assets := os.DirFS("testdata/basic/assets")
	// site := powerlinx.NewSite(content, templates, powerlinx.IncludeDrafts())
	site, err := powerlinx.NewSite(content, templates)
	if err != nil {
		panic(err)
	}

	log.Println("Generating Site in ./pub")
	err = site.GenerateSite("testdata/basic/public")
	if err != nil {
		panic(err)
	}

	fileserver := http.FileServer(HTMLDir{http.Dir("testdata/basic/public/")})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assets))))
	http.Handle("/", http.StripPrefix("/", fileserver))

	log.Fatal((http.ListenAndServe(":8080", nil)))
}

func main() {
	Basic()
}
