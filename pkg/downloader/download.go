package downloader

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/html"
)

const (
	dumpsURLfmt = `https://dumps.wikimedia.org/%swiki/latest/`
)

func DownloadDumps(basefolder string, lang string, files []string) (chan string, error) {
	// Size 1 to throttle download to inserter use, only downloading the next dump so insert never wait
	// this way, in tight mode, only 2 dump will be on disk at any given time
	filech := make(chan string, 1)

	fmt.Printf("Using %d dump files:\n", len(files))
	for _, filename := range files {
		fmt.Printf("- %s\n", filename)
	}

	go func() {
		err := download(basefolder, files, lang, filech)
		if err != nil {
			fmt.Printf("DownloadDumps error: %s\n", err)
		}
		close(filech)
	}()

	return filech, nil
}

func download(basefolder string, files []string, lang string, ch chan string) error {
	for _, filename := range files {
		// Is dump extracted already ? if so send filename
		extractFilename := strings.TrimSuffix(filename, ".bz2")
		extracted := fileExists(path.Join(basefolder, extractFilename))
		if extracted {
			fmt.Printf("Found %s at %s\n", extractFilename, path.Join(basefolder, extractFilename))
			ch <- extractFilename
			continue
		}

		// Is dump here already ? if not download
		exist := fileExists(path.Join(basefolder, filename))
		if !exist {
			fmt.Printf("Downloading %s\n", filename)
			err := downloadDump(basefolder, filename, lang)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("Found %s at %s\n", filename, path.Join(basefolder, filename))
		}

		// Is dump extracted already ? if not extract
		extractFilename = strings.TrimSuffix(filename, ".bz2")
		extracted = fileExists(path.Join(basefolder, extractFilename))
		if !extracted {
			fmt.Printf("Extracting %s\n", filename)
			begin := time.Now()
			err := extractDump(basefolder, filename, extractFilename)
			if err != nil {
				return err
			}
			fmt.Printf("Extracted %s (took %s)\n", filename, time.Since(begin))
		} else {
			fmt.Printf("Found %s at %s\n", extractFilename, path.Join(basefolder, extractFilename))
		}

		ch <- extractFilename
	}
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func extractDump(basefolder, filename string, extractFilename string) error {
	cmd := exec.Command("bzip2", "-d", filename)
	cmd.Dir = basefolder
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	log.Debugf("Exec %+v\n", cmd.Args)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("wget output: '%s'\n", out.String())
		return err
	}

	return nil
}

func downloadDump(basefolder, filename, lang string) error {
	url := fmt.Sprintf(dumpsURLfmt, lang) + filename
	cmd := exec.Command("wget", url, "-O", path.Join(basefolder, filename))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	log.Debugf("Exec %+v\n", cmd.Args)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("wget output: '%s'\n", out.String())
		return err
	}

	return nil
}

func ListArticleDumps(interactive bool, lang string) ([]string, error) {
	var urls []string

	dumpsURL := fmt.Sprintf(dumpsURLfmt, lang)
	resp, err := http.Get(dumpsURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: %s", dumpsURL, resp.Status)
	}
	defer resp.Body.Close()

	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			if interactive {
				return selectFiles(urls), nil
			}
			return urls, nil
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			// Extract the href value, if there is one
			ok, url := getHref(t)
			if !ok {
				continue
			}

			// Make sure the url contains "article" and has ".xml.gz" suffix
			if strings.Contains(url, "pages-articles-multistream") &&
				strings.Contains(url, ".xml-") &&
				strings.HasSuffix(url, ".bz2") {
				urls = append(urls, url)
			}

		}
	}
}

func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	return
}

func selectFiles(url []string) []string {
	var selected []string

	reader := bufio.NewReader(os.Stdin)
	for _, u := range url {
		fmt.Printf("Import %s [Y/n]: ", u)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		if text != "n" {
			selected = append(selected, u)
		}
	}

	return selected
}
