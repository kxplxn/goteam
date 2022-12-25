package db

import (
	"database/sql"
)

// CreatorSession manages active sessions and provides access to them.
type CreatorSession struct{ db *sql.DB }

// NewCreatorSession is the constructor for CreatorSession.
func NewCreatorSession(db *sql.DB) *CreatorSession {
	return &CreatorSession{db: db}
}

// Create creates a session in the database with the given id, username, and
// expiry.
func (m *CreatorSession) Create(args ...any) error {
	_, err := m.db.Exec(
		`INSERT INTO sessions(id, username, expiry) VALUES ($1, $2, $3)`,
		args,
	)
	return err
}
