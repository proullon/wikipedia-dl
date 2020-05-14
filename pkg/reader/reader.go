package reader

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Dump struct {
	Info  SiteInfo `xml:"siteinfo"`
	Pages []Page   `xml:"page"`
}

type SiteInfo struct {
	SiteName string `xml:"sitename"`
	DBName   string `xml:"dbname"`
}

type Page struct {
	Title string `xml:"title"`
	ID    int    `xml:"id"`
	Text  string `xml:"revision>text"`
}

type Reader struct {
	dirname string
	index   int
	files   []string
}

func New(dirname string) (*Reader, error) {
	r := &Reader{
		dirname: dirname,
	}

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".xml") {
			r.files = append(r.files, f.Name())
		}
	}

	return r, nil
}

func (r *Reader) Count() int {
	return len(r.files)
}

func (r *Reader) Next() (*Dump, error) {
	if r.index >= len(r.files) {
		return nil, io.EOF
	}

	d, err := ReadDump(path.Join(r.dirname, r.files[r.index]))
	r.index++
	return d, err
}

func ReadDump(filename string) (*Dump, error) {
	fmt.Printf("Reading %s\n", filename)

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	d := &Dump{}
	err = xml.Unmarshal(data, d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func StreamDumpPages(filename string) (*SiteInfo, chan Page, error) {
	fmt.Printf("Reading %s\n", filename)

	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}

	decoder := xml.NewDecoder(f)

	_, err = decoder.Token()
	if err != nil {
		return nil, nil, fmt.Errorf("xml.Token: %s", err)
	}

	si := &SiteInfo{}
	err = decoder.Decode(si)
	if err != nil {
		return nil, nil, fmt.Errorf("xml.DecodeElement(siteinfo): %s", err)
	}

	pchan := make(chan Page, 10)

	go func() {
		defer close(pchan)
		defer f.Close()

		for {

			p := Page{}
			err = decoder.Decode(&p)
			if err != nil {
				log.Infof("DecodeElement: %d", err)
				return
			}

			pchan <- p
		}
	}()

	return si, pchan, nil
}
