/*** Liveness probe for API server. It detects if etcd was restored from backup. ***/

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
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
		os.Exit(1)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "etcdbr_restoration_duration_seconds_count{succeeded=\"true\"}") {
			current := strings.Split(line, " ")

			if _, err := os.Stat(LastFile); os.IsExist(err) {
				data, err := ioutil.ReadFile(LastFile)
				if err != nil {
					log.Fatalf("Could not read file:", err)
					os.Exit(1)
				}

				last, err := strconv.Atoi(string(data))
				if err != nil {
					log.Fatalf("Error converting last:", err)
					os.Exit(1)
				}

				currentInt, err := strconv.Atoi(current[1])
				if err != nil {
					log.Fatalf("Error converting current:", err)
					os.Exit(1)
				}

				if last < currentInt {
					log.Fatalf("Etcd restore detected.")
					os.Exit(1)
				}
			}

			err = ioutil.WriteFile(LastFile, []byte(current[1]), 0644)
			if err != nil {
				log.Fatalf("Could not write file:", err)
				os.Exit(1)
			}
		}
	}
	// exit 0
}
