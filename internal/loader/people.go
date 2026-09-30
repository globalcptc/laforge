package loader

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
)

// loadPeopleCSV reads one people/*.csv file per "A people CSV is data, not
// configuration": the header row's columns become each Person's attribute
// bag (except "username", which is pulled out as its own field since every
// other object references people by it -- see "people: [arivera3] on any
// object"). No fixed column set is assumed beyond that.
func (c *Content) loadPeopleCSV(rel, path string) {
	f, err := os.Open(path)
	if err != nil {
		c.addf(rel, 0, "read people CSV: %v", err)
		return
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		c.addf(rel, 0, "parse CSV: %v", err)
		return
	}
	if len(rows) == 0 {
		c.addf(rel, 0, "people CSV is empty, expected at least a header row")
		return
	}

	header := rows[0]
	usernameCol := -1
	for i, h := range header {
		if strings.EqualFold(strings.TrimSpace(h), "username") {
			usernameCol = i
			break
		}
	}
	if usernameCol == -1 {
		c.addf(rel, 1, "people CSV has no 'username' column -- every object that references a person does so by username")
		return
	}

	src := PeopleSource{
		SourceFile: rel,
		Name:       strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)),
	}

	seen := make(map[string]int) // username -> row number, for collision reporting
	for i, row := range rows[1:] {
		lineNo := i + 2 // +1 for header, +1 because rows[1:] is 0-indexed
		if len(row) != len(header) {
			c.addf(rel, lineNo, "row has %d columns, header has %d", len(row), len(header))
			continue
		}
		username := strings.TrimSpace(row[usernameCol])
		if username == "" {
			c.addf(rel, lineNo, "row has an empty username")
			continue
		}
		if firstLine, dup := seen[username]; dup {
			c.addf(rel, lineNo, "duplicate username %q (first seen on line %d)", username, firstLine)
			continue
		}
		seen[username] = lineNo

		p := Person{SourceFile: rel, Username: username, Attributes: make(map[string]string, len(header))}
		for col, h := range header {
			if col == usernameCol {
				continue
			}
			p.Attributes[strings.TrimSpace(h)] = row[col]
		}
		src.People = append(src.People, p)
	}

	c.People = append(c.People, src)
}
