package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Strips file extension off of the file and prepends a /
func FilepathToUrl(file string) (string, string) {
	ext := filepath.Ext(file)
	return strings.TrimSuffix("/"+file, ext), ext
}

func getAllPagesInDir(dir string, pages []string) []string {
	// TODO: check whether to include subdirs or not
	matchingPages := []string{}
	for _, url := range pages {
		if strings.HasPrefix(url, dir) && path.Base(url) != "index" {
			matchingPages = append(matchingPages, url)
		}
	}
	return matchingPages
}

func convertMdToHTML(data []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := markdown.Convert(data, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func parseMetadata(metadata []byte) (PageMetadata, error) {
	pagemeta := PageMetadata{}
	if len(metadata) > 0 {
		err := json.Unmarshal(metadata, &pagemeta)
		if err != nil {
			return pagemeta, err
		}
	}
	return pagemeta, nil
}

func createFile(outPath string) (*os.File, error) {
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

func writePage(outPath string, page Page) error {
	outFile, err := createFile(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	fileWriter := bufio.NewWriter(outFile)
	err = page.Render(fileWriter)
	if err != nil {
		return err
	}
	err = fileWriter.Flush()
	if err != nil {
		return err
	}
	return nil
}
