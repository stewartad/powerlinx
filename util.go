package powerlinx

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	matchingPages := []string{}
	for _, url := range pages {
		if strings.HasPrefix(url, dir) && path.Base(url) != "index" {
			matchingPages = append(matchingPages, url)
		}
	}
	return matchingPages
}

func convertMdToHTML(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := markdown.Convert(data, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseMetadata(metadata []byte) (PageMetadata, error) {
	bytes.TrimSpace(metadata)
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

func writeFeed(outPath string, data string) error {
	outFile, err := createFile(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	fileWriter := bufio.NewWriter(outFile)
	_, err = fileWriter.WriteString(data)
	if err != nil {
		return err
	}
	err = fileWriter.Flush()
	if err != nil {
		return err
	}
	return nil
}

// deletes the target directory then creates an empty one in its place
func recreateDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.Mkdir(dir, 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}
