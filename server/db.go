package main

import (
	"database/sql"
	"strconv"
	"strings"
)

// DB and Tx wrap database/sql so the existing queries written with `?`
// placeholders run unchanged on PostgreSQL, which expects $1, $2, ...
type DB struct{ *sql.DB }
type Tx struct{ *sql.Tx }

// rebind converts `?` placeholders to $n, skipping anything inside single-quoted string literals.
func rebind(q string) string {
	if !strings.Contains(q, "?") {
		return q
	}
	var b strings.Builder
	b.Grow(len(q) + 16)
	n, inString := 0, false
	for i := 0; i < len(q); i++ {
		ch := q[i]
		if ch == '\'' {
			inString = !inString
		}
		if ch == '?' && !inString {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func (d *DB) Exec(q string, args ...any) (sql.Result, error) { return d.DB.Exec(rebind(q), args...) }
func (d *DB) Query(q string, args ...any) (*sql.Rows, error) { return d.DB.Query(rebind(q), args...) }
func (d *DB) QueryRow(q string, args ...any) *sql.Row        { return d.DB.QueryRow(rebind(q), args...) }
func (d *DB) Begin() (*Tx, error) {
	t, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{t}, nil
}

func (t *Tx) Exec(q string, args ...any) (sql.Result, error) { return t.Tx.Exec(rebind(q), args...) }
func (t *Tx) Query(q string, args ...any) (*sql.Rows, error) { return t.Tx.Query(rebind(q), args...) }
func (t *Tx) QueryRow(q string, args ...any) *sql.Row        { return t.Tx.QueryRow(rebind(q), args...) }
