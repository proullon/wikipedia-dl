package inserter

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/proullon/workerpool"
	log "github.com/sirupsen/logrus"

	"github.com/proullon/wikipedia-to-cockroachdb/pkg/parser"
	"github.com/proullon/wikipedia-to-cockroachdb/pkg/reader"
)

type Inserter struct {
	errch chan error

	db                   *sql.DB
	insertPageContent    bool
	insertPageReferences bool
	done                 int
	errors               int

	wp *workerpool.WorkerPool
}

func New(db *sql.DB, n int, insertPageContent bool, insertPageReferences bool) *Inserter {
	i := &Inserter{
		errch:                make(chan error),
		db:                   db,
		insertPageContent:    insertPageContent,
		insertPageReferences: insertPageReferences,
	}

	i.wp, _ = workerpool.New(i.Insert,
		workerpool.WithRetry(15),
		workerpool.WithMaxWorker(n),
		workerpool.WithMaxQueue(1000),
		workerpool.WithSizePercentil(workerpool.AllSizesPercentil),
	)
	return i
}

func (i *Inserter) ImportStream(pagech chan reader.Page) chan error {

	go func() {
		for p := range pagech {
			i.wp.Feed(p)
		}
		log.Infof("ImportStream: Done feeding WorkerPool")
		i.wp.Wait()
		i.wp.Stop()

		v := i.wp.VelocityValues()
		fmt.Printf("Velocity:\n")
		for i := 1; i <= 100; i++ {
			velocity, ok := v[i]
			if !ok {
				continue
			}
			fmt.Printf("%d%% : %fop/s\n", i, velocity)
		}

		percentil, ops := i.wp.CurrentVelocityValues()
		fmt.Printf("Current velocity: %d%% -> %f op/s\n", percentil, ops)
	}()

	go func() {
		for r := range i.wp.Responses() {
			i.done++
			if r.Err != nil {
				i.errors++
				i.errch <- r.Err
			}
		}
		close(i.errch)
	}()

	go func() {
		for {
			if i.wp.Status() == workerpool.Stopped {
				return
			}
			time.Sleep(10 * time.Second)
			percentil, ops := i.wp.CurrentVelocityValues()
			log.Infof("%d articles done (%d errors). Current velocity %d%% (%f op/s) (%d cached, %d hits)\n", i.done, i.errors, percentil, ops, Cached(), hit)
		}
	}()

	return i.errch
}

func (i *Inserter) Insert(payload interface{}) (interface{}, error) {
	p := payload.(reader.Page)
	err := i.insert(p)
	if err != nil {
		return nil, err
	}

	return true, nil
}

func (i *Inserter) insert(p reader.Page) error {
	var err error

	// TODO: should be in config
	var ignoredPrefixes = []string{
		"wikipedia", "template", "project", "portal", "category", "draft", "module",
		"wikipédia", "modèle", "projet", "portail", "catégorie", "draft", "module",
	}

	// do not insert wikipedia meta page
	for _, prefix := range ignoredPrefixes {
		if strings.HasPrefix(strings.ToLower(p.Title), prefix+":") {
			log.Infof("Ignoring %s", p.Title)
			return nil
		}
	}

	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): Begin : %s", p.Title, p.ID, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `DELETE FROM page WHERE page_id = $1`
	_, err = tx.Exec(query, p.ID)
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): DELETE : %s", p.Title, p.ID, err)
	}

	query = `INSERT INTO page (page_id, title, lower_title) VALUES ($1, $2, $3)`
	_, err = tx.Exec(query, p.ID, p.Title, strings.ToLower(p.Title))
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): INSERT page : %s", p.Title, p.ID, err)
	}

	if i.insertPageContent {
		err = insertPageContent(tx, &p)
		if err != nil {
			return err
		}
	}

	if i.insertPageReferences {
		err = insertPageReferences(i.db, tx, &p)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): COMMIT : %s", p.Title, p.ID, err)
	}

	return nil
}

func insertPageContent(tx *sql.Tx, p *reader.Page) error {
	query := `DELETE FROM page_content WHERE page_id = $1`
	_, err := tx.Exec(query, p.ID)
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): DELETE : %s", p.Title, p.ID, err)
	}

	query = `INSERT INTO page_content (page_id, content) VALUES ($1, $2)`
	_, err = tx.Exec(query, p.ID, p.Text)
	if err != nil {
		return fmt.Errorf("Inserting %s (%d): INSERT page_content : %s", p.Title, p.ID, err)
	}

	return nil
}

func insertPageReferences(db *sql.DB, tx *sql.Tx, p *reader.Page) error {

	references := parser.PageReferences(p)

	query := `DELETE FROM article_reference WHERE page_id = $1`
	_, err := tx.Exec(query, p.ID)
	if err != nil {
		return err
	}

	// no references to import, thank you next
	if len(references) == 0 {
		return nil
	}

	var refID int
	var first bool = true
	query = `INSERT INTO article_reference (page_id, refered_page, occurrence, reference_index) VALUES `
	existingReferences := make(map[int]*parser.Reference)
	for _, ref := range references {
		r := ref.Title

		refID, err = GetPage(db, r)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			log.Errorf("Cannot find page '%s': %s\n", r, err)
			continue
		}

		eref, ok := existingReferences[refID]
		if ok {
			eref.Occurence += ref.Occurence
		} else {
			ref.ID = refID
			existingReferences[refID] = ref
		}

	}

	for _, ref := range existingReferences {
		if first {
			query = fmt.Sprintf("%s (%d, %d, %d, %d) ", query, p.ID, ref.ID, ref.Occurence, ref.Index)
			first = false
		} else {
			query = fmt.Sprintf("%s, (%d, %d, %d, %d) ", query, p.ID, ref.ID, ref.Occurence, ref.Index)
		}
	}

	// first is true so all GetPage failed :(
	if first == true {
		return nil
	}

	_, err = tx.Exec(query)
	if err != nil {
		return fmt.Errorf("%s: || %s || %s", p.Title, query, err)
	}

	return nil
}
