package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const schema = `CREATE TABLE IF NOT EXISTS %s
(
  key text PRIMARY KEY,
  value text NOT NULL,
  valid_until timestamp without time zone NOT NULL
);`

var db *sql.DB

func newCache(cleanupInterval time.Duration) (c *cache, err error) {
	dbname := os.Getenv("DBNAME")
	if dbname == "" {
		dbname = "keystone-cache"
	}
	dbuser := os.Getenv("DBUSER")
	dbhost := os.Getenv("DBHOST")
	if dbhost == "" {
		dbhost = "localhost"
	}

	db, err = sql.Open("postgres", fmt.Sprintf("dbname=%s host=%s sslmode=disable user=%s", dbname, dbhost, dbuser))
	if err != nil {
		return
	}
	if err = db.Ping(); err != nil {
		return
	}

	//ensure schema exists
	db.Exec(fmt.Sprintf(schema, "token_cache"))
	c = New(db, cleanupInterval, "token_cache").(*cache)
	return

}

func TestCache(t *testing.T) {
	c, err := newCache(30 * time.Second)
	if err != nil {
		t.Fatal("Failed to create cache", err)
	}
	testkey := fmt.Sprintf("%d", time.Now().UnixNano())
	c.Set(testkey, "blafasel", 1*time.Minute)

	var value string
	if ok := c.Get(testkey, &value); !ok || value != "blafasel" {
		t.Fatalf("Expected %q, got %q", "blafasel", value)
	}

}

func TestCacheExpiry(t *testing.T) {
	c, err := newCache(30 * time.Second)
	if err != nil {
		t.Fatal("Failed to create cache", err)
	}
	c.Set("expire", "blafasel", 20*time.Millisecond)
	<-time.After(25 * time.Millisecond)
	var output string
	if c.Get("expire", &output) {
		t.Error("Key found that should have been expired")
	}
}

func TestCleanup(t *testing.T) {
	c, err := newCache(10 * time.Millisecond)
	if err != nil {
		t.Fatal("Failed to create cache", err)
	}
	c.Set("cleanup", "blafasel", 20*time.Millisecond)
	c.Set("stay", "blafasel", 1*time.Second)
	<-time.After(50 * time.Millisecond)
	var val string
	if db.QueryRow("SELECT value from token_cache WHERE key=$1", "cleanup").Scan(&val) == nil {
		t.Error("Row found which should have been cleaned up")
	}

	if db.QueryRow("SELECT value from token_cache WHERE key=$1", "stay").Scan(&val) != nil {
		t.Error("No row found for valid entry which should not have been deleted")
	}

}
