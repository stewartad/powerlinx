package powerlinx

import (
	"html/template"
	"io"
	"log"
)

// View stores information about a template
type View struct {
	Template *template.Template
	Layout   string
}

// Execute a given template, with the page contents passed in data
func (v *View) Render(w io.Writer, data interface{}) error {
	return v.Template.ExecuteTemplate(w, v.Layout, data)
}

// Add a View to the Site
func (s *Site) NewView(layout string, lastTmpl string) *View {
	t, err := template.ParseFS(s.templates, lastTmpl, "base/*.html")
	if err != nil {
		log.Fatal(err)
	}

	return &View{Template: t, Layout: layout}
}
