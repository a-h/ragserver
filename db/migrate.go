package db

import (
	"embed"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/rqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
)

func ParseRqliteURL(s string) (u RqliteURL, err error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return u, fmt.Errorf("db: parse rqlite URL failed: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return u, fmt.Errorf("db: parse rqlite URL failed: invalid scheme %q", parsed.Scheme)
	}
	if parsed.Port() == "" {
		parsed.Host = fmt.Sprintf("%s:4001", parsed.Hostname())
	}
	return RqliteURL{URL: parsed}, nil
}

type RqliteURL struct {
	URL *url.URL
}

func (ru RqliteURL) DataSourceName() string {
	return ru.URL.String()
}

func (ru RqliteURL) MigrateDatabaseURL() string {
	u := &url.URL{
		Scheme: "rqlite",
		User:   ru.URL.User,
		Host:   fmt.Sprintf("%s:%s", ru.URL.Hostname(), ru.URL.Port()),
	}
	if ru.URL.Scheme == "http" {
		q := u.Query()
		q.Set("x-connect-insecure", "true")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

//go:embed migrations/*.sql
var fs embed.FS

func Migrate(u RqliteURL) (err error) {
	srcDriver, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("db: migrate failed to create iofs: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", srcDriver, u.MigrateDatabaseURL())
	if err != nil {
		return fmt.Errorf("db: migrate failed to create source instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("db: migrate up failed: %w", err)
	}
	return nil
}
