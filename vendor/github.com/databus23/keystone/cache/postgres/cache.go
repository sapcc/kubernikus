//Package postgres provides a postgres backed cache implementation for https://github.com/databus23/keystone
package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/databus23/keystone"
)

type pgCache struct {
	db      *sql.DB
	table   string
	janitor *janitor
}

// New creates a new cache.
//
// The table parameter defaults to token_cache and must point to an existing datbase table conforming to the following schema:
//  key text PRIMARY KEY,
//  value text NOT NULL,
//  valid_until timestamp without time zone NOT NULL
func New(db *sql.DB, cleanupInterval time.Duration, table string) keystone.Cache {
	if table == "" {
		table = "token_cache"
	}

	c := &pgCache{db: db, table: table}
	//see new to understand why this exists
	cache := &cache{c}
	runJanitor(c, cleanupInterval)
	runtime.SetFinalizer(cache, stopJanitor)
	return cache
}

func (s *pgCache) Set(key string, x interface{}, ttl time.Duration) {
	if b, err := json.Marshal(x); err == nil {
		tx, err := s.db.Begin()
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()

		if _, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE key=$1`, s.table), key); err != nil {
			keystone.Log("Failed to delete: %v", err)
			return
		}
		if _, err = tx.Exec(fmt.Sprintf(`INSERT INTO "%s" (key,value,valid_until) VALUES ($1,$2,$3)`, s.table), key, string(b), time.Now().Add(ttl)); err != nil {
			keystone.Log("Failed to insert: %v", err)
			return
		}
	}
}

func (s *pgCache) Get(k string, x interface{}) bool {
	var data string
	if err := s.db.QueryRow(fmt.Sprintf(`SELECT value FROM "%s" WHERE key=$1 AND now() < valid_until`, s.table), k).Scan(&data); err != nil {
		return false
	}
	if json.Unmarshal([]byte(data), x) != nil {
		return false
	}
	return true
}

func (s *pgCache) deleteExpired() {
	s.db.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE valid_until < now()`, s.table))
}

//taken from https://github.com/pmylund/go-cache/blob/master/cache.go
//this struct is a wrapper to allow the gc to stop the janitor go routine
type cache struct {
	*pgCache
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *pgCache) {
	j.stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func runJanitor(c *pgCache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
	}
	c.janitor = j
	go j.Run(c)
}

func stopJanitor(c *cache) {
	c.janitor.stop <- true
}
