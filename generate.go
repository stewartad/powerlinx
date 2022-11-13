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

func writePage(outFile *os.File, page Page) error {
	fileWriter := bufio.NewWriter(outFile)
	err := page.Render(fileWriter)
	if err != nil {
		return err
	}
	err = fileWriter.Flush()
	if err != nil {
		return err
	}
	return nil
}

func (s *Site) GenerateSite(outdir string) error {
	err := os.RemoveAll(outdir)
	if err != nil {
		log.Println("could not delete pub")
	}

	err = os.Mkdir(outdir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
	}

	// write out aggregate pages
	for dir, page := range s.ListPageMap {
		outPath := path.Clean(path.Join(outdir + dir + string(os.PathSeparator) + "index.html"))
		outFile, err := createHTMLFile(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		// err = writeListPage(outFile, page)
		err = writePage(outFile, page)
		if err != nil {
			return err
		}
	}

	// write out single pages, overwriting any generated aggregate pages that have custom implementation
	for url, page := range s.PageMap {
		outPath := path.Join(outdir + url + ".html")
		outFile, err := createHTMLFile(outPath)
		if err != nil {
			return err
		}
		defer outFile.Close()
		// err = writeSinglePage(outFile, page)
		err = writePage(outFile, page)
		if err != nil {
			return err
		}
	}
	return nil
}
