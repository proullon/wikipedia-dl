package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/proullon/wikipedia-to-cockroachdb/pkg/downloader"
	"github.com/proullon/wikipedia-to-cockroachdb/pkg/inserter"
	"github.com/proullon/wikipedia-to-cockroachdb/pkg/reader"
)

func Import(db *sql.DB, basefolder string, parallelisationFactor int, tightmode bool, withPageContent bool, withPageReferences bool, interactive bool, language string) error {

	urls, err := downloader.ListArticleDumps(interactive, language)
	if err != nil {
		return err
	}
	filech, err := downloader.DownloadDumps(basefolder, language, urls)
	if err != nil {
		return err
	}

	for dumpName := range filech {
		p := path.Join(basefolder, dumpName)
		fmt.Printf("Opening %s\n", p)
		begin := time.Now()

		si, pagech, err := reader.StreamDumpPages(p)
		if err != nil {
			return err
		}

		fmt.Printf("Inserting dump %s: %+v (opening took %s)\n", dumpName, si, time.Since(begin))
		begin = time.Now()
		i := inserter.New(db, parallelisationFactor, withPageContent, withPageReferences)

		errch := i.ImportStream(pagech)
		var errc int
		for err := range errch {
			log.Errorf("%s: %s", dumpName, err)
			errc++
		}

		fmt.Printf("Finished %s done (%s) (%d errors)\n", dumpName, time.Since(begin), errc)

		if tightmode {
			err = removeDump(path.Join(basefolder, dumpName))
			if err != nil {
				log.Errorf("cannot remove file %s: %s", dumpName, err)
			}
			err = removeDumpArchive(path.Join(basefolder, dumpName))
			if err != nil {
				log.Errorf("cannot remove archive file %s: %s", dumpName, err)
			}
		}
	}

	return nil
}

func removeDump(filepath string) error {
	fmt.Printf("Removing %s\n", filepath)
	return os.Remove(filepath)
}

func removeDumpArchive(filepath string) error {
	filepath += ".bz2"
	fmt.Printf("Removing %s\n", filepath)
	return os.Remove(filepath)
}
