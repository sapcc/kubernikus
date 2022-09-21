/*** Liveness probe for API server. It detects if etcd was restored from backup. ***/

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	LastFile = "/tmp/last"
)

func main() {
	if os.Getenv("ETCD_HOST") == "" || os.Getenv("ETCD_BACKUP_PORT") == "" {
		log.Fatal("Environment variables ETCD_HOST and ETCD_BACKUP_PORT expected. Exiting.")
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%s/metrics", os.Getenv("ETCD_HOST"), os.Getenv("ETCD_BACKUP_PORT"))
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Could not reach URL: %s", err)
	}
	defer resp.Body.Close()

	current := 0
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "etcdbr_restoration_duration_seconds_count{succeeded=\"true\"}") {
			c := strings.Split(line, " ")
			if current, err = strconv.Atoi(c[1]); err != nil {
				log.Fatalf("Error converting current: %s", err)
			}
			break
		}
	}

	if _, err := os.Stat(LastFile); err == nil {
		data, err := os.ReadFile(LastFile)
		if err != nil {
			log.Fatalf("Could not read file: %s", err)
		}
		last, err := strconv.Atoi(string(data))
		if err != nil {
			log.Fatalf("Error converting last: %s", err)
		}
		if last != current {
			log.Fatal("Etcd restore detected")
		}
	} else {
		if err := os.WriteFile(LastFile, []byte(strconv.Itoa(current)), 0644); err != nil {
			log.Fatalf("Failed to write file: %s", err)
		}
	}
	// exit 0
}
