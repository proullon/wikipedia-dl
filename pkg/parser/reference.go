package parser

import (
	"regexp"
	"strings"

	"github.com/proullon/wikipedia-to-cockroachdb/pkg/reader"
)

type Reference struct {
	ID        int
	Title     string
	Occurence int
	Index     int
}

func Cleanup(s string) string {
	s = strings.TrimPrefix(s, "[[")
	s = strings.TrimSuffix(s, "]]")

	s = strings.Split(s, "|")[0]
	s = strings.Split(s, "#")[0]

	t := strings.Split(s, ":")
	if len(t) != 1 {
		s = t[1]
	}

	s = strings.TrimLeft(s, " ")
	s = strings.TrimRight(s, " ")

	s = strings.Replace(s, "_", " ", -1)

	s = strings.ToLower(s)

	return s
}

func PageReferences(p *reader.Page) map[string]*Reference {
	c := p.Text

	// Try to stop before '==See also=='
	i := strings.Index(c, "==See also==")
	if i > 0 {
		//fmt.Printf("Found ==See also== (%d)!\n", i)
		c = c[:i]
	}
	i = strings.Index(c, "== See also ==")
	if i > 0 {
		//fmt.Printf("Found == See also == (%d)!\n", i)
		c = c[:i]
	}

	// TODO: should be in config
	var ignoredPrefixes = []string{
		"wikipedia", "template", "project", "portal", "category", "draft", "module", "list",
		"wikipédia", "modèle", "projet", "portail", "catégorie", "ébauche", "module", "liste",
	}

	re := regexp.MustCompile(`\[\[.(.*?)\]\]`)
	sub := re.FindAllString(c, -1)

	var index int
	references := make(map[string]*Reference)
	for _, s := range sub {

		// do not insert wikipedia meta page
		for _, prefix := range ignoredPrefixes {
			if strings.HasPrefix(strings.ToLower(s), prefix+":") {
				continue
			}
		}

		s = Cleanup(s)

		if s == "" {
			continue
		}

		ref, ok := references[s]
		if ok {
			ref.Occurence++
		} else {
			index++
			references[s] = &Reference{
				Title:     s,
				Occurence: 1,
				Index:     index,
			}
		}
	}
	return references
}

func IsList(p *reader.Page) bool {
	if strings.HasPrefix(strings.ToLower(p.Title), "list") {
		return true
	}

	return false
}

func IsLanguage(p *reader.Page) bool {
	d := p.Text

	re := regexp.MustCompile(`{{Infobox(.*?)language`)
	if re.MatchString(d) {
		return true
	}

	return false
}

func IsHuman(p *reader.Page) bool {
	d := p.Text

	re := regexp.MustCompile(`{{Infobox(.*?)scientist`)
	if re.MatchString(d) {
		return true
	}

	re = regexp.MustCompile(`{{Infobox(.*?)artist`)
	if re.MatchString(d) {
		return true
	}

	if strings.Contains(d, "| birth_date") || strings.Contains(d, "|birth_date") {
		return true
	}

	return false
}

func IsPlace(p *reader.Page) bool {
	d := p.Text

	re := regexp.MustCompile(`{{Infobox(.*?)commune`)
	if re.MatchString(d) {
		return true
	}
	re = regexp.MustCompile(`{{Infobox(.*?)town`)
	if re.MatchString(d) {
		return true
	}
	re = regexp.MustCompile(`{{Infobox(.*?)country`)
	if re.MatchString(d) {
		return true
	}
	re = regexp.MustCompile(`{{Infobox(.*?)state`)
	if re.MatchString(d) {
		return true
	}
	re = regexp.MustCompile(`{{Infobox(.*?)settlement`)
	if re.MatchString(d) {
		return true
	}
	/*
		if strings.Contains(p.Revisions[0].Content, "|coordinate") ||
			strings.Contains(p.Revisions[0].Content, "|Coordinate") ||
			strings.Contains(p.Revisions[0].Content, "| coordinate") ||
			strings.Contains(p.Revisions[0].Content, "| Coordinate") ||
			strings.Contains(p.Revisions[0].Content, "|Latitude") ||
			strings.Contains(p.Revisions[0].Content, "|latitude") ||
			strings.Contains(p.Revisions[0].Content, "| latitude") ||
			strings.Contains(p.Revisions[0].Content, "| Latitude") {
			return true
		}
	*/
	return false
}
