package inserter

import (
	"database/sql"
	"strings"
	"sync"
)

// we could use redis, but a MAP is enough
var (
	PageIndex map[string]int
	indexm    sync.Mutex
	hit       int
)

func initialize() {
	PageIndex = make(map[string]int)
}

func Cached() int {
	indexm.Lock()
	n := len(PageIndex)
	indexm.Unlock()
	return n
}

func Cache(title string, id int) {
	indexm.Lock()
	PageIndex[title] = id
	indexm.Unlock()
}

func GetPage(db *sql.DB, title string) (int, error) {
	title = strings.ToLower(title)

	indexm.Lock()
	if PageIndex == nil {
		PageIndex = make(map[string]int)
	}
	id, ok := PageIndex[title]
	indexm.Unlock()
	if ok {
		hit++
		return id, nil
	}

	query := `SELECT page_id FROM page WHERE lower_title = $1`
	err := db.QueryRow(query, title).Scan(&id)
	if err != nil {
		return 0, err
	}

	indexm.Lock()
	PageIndex[title] = id
	indexm.Unlock()
	return id, nil
}
