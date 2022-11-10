package powerlinx

import (
	"bufio"
	"log"
	"os"
	"path"
)

func createHTMLFile(outPath string) (*os.File, error) {
	err := os.MkdirAll(path.Dir(outPath), 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *Site) GenerateSite() {
	err := os.RemoveAll("pub")
	if err != nil {
		log.Println("could not delete pub")
	}

	err = os.Mkdir("pub", 0755)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
	}

	for url, page := range s.PageMap {
		outPath := path.Join("pub" + url + ".html")
		outFile, err := createHTMLFile(outPath)
		if err != nil {
			// TODO: better handling
			panic(err)
		}

		fileWriter := bufio.NewWriter(outFile)
		// TODO: determine the real template to render
		// TODO: properly generate blog page

		err = page.View.Render(fileWriter, page)
		if err != nil {
			panic(err)
		}
		err = fileWriter.Flush()
		if err != nil {
			panic(err)
		}
	}
}
