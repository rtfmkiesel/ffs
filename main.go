//go:build linux

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// To find the default profile in profiles.ini
	reDefaultProfile = regexp.MustCompile(`\[Install.*\]`)
)

const (
	// Where the db is copied to temporarily
	dbTmpPath = "/tmp/places.sqlite"
	// The SQL query to get the history filtered by the argument as a glob pattern
	histQuery = `
		SELECT DISTINCT url
		FROM moz_places
		JOIN moz_historyvisits ON moz_places.id = moz_historyvisits.place_id
		WHERE LOWER(url) GLOB LOWER(?) OR LOWER(title) GLOB LOWER(?) OR LOWER(description) GLOB LOWER(?)
		ORDER BY last_visit_date ASC`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: ffs \"<query>\"\n")
		os.Exit(1)
	}

	query := os.Args[1]
	if query == "" {
		fmt.Fprintf(os.Stderr, "usage: ffs \"<query>\"\n")
		os.Exit(1)
	}

	// Get the Firefox profile dir
	profileDir, err := getFirefoxProfileDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get Mozilla profile directory: %s\n", err)
		os.Exit(1)
	}
	dbPath := profileDir + "/places.sqlite"

	// Copy the db to /tmp to avoid running into locks
	if err := copyFile(dbPath, dbTmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	defer os.Remove(dbTmpPath)

	// Open the db
	db, err := sql.Open("sqlite3", dbTmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Prepare the query
	pattern := convertToGlobPattern(query)
	params := []interface{}{pattern, pattern, pattern}

	// Execute the query
	rows, err := db.Query(histQuery, params...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	// To track printed results
	printedUrls := make(map[string]bool)

	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			fmt.Fprintf(os.Stderr, "error scanning row: %s\n", err)
			continue
		}

		// Do not print if already printed
		if _, ok := printedUrls[url]; ok {
			continue
		}

		printedUrls[url] = true
		fmt.Println(url)
	}

	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error iterating rows: %s\n", err)
		os.Exit(1)
	}
}

// Returns the currently default Mozilla Firefox profile directory
func getFirefoxProfileDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %s", err)
	}

	ffdir := homeDir + "/.mozilla/firefox"
	profileDir, err := parseProfileIni(ffdir)
	if err != nil {
		return "", err
	}

	return ffdir + "/" + profileDir, nil
}

// Parses the profiles.ini file to get the default Firefox profile
func parseProfileIni(ffdir string) (string, error) {
	iniPath := ffdir + "/profiles.ini"
	iniFh, err := os.Open(iniPath)
	if err != nil {
		return "", fmt.Errorf("could not open profiles.ini: %s", err)
	}
	defer iniFh.Close()

	// Read line by line until the regex matches
	inDefaultProfile := false
	scanner := bufio.NewScanner(iniFh)
	for scanner.Scan() {
		line := scanner.Text()
		if reDefaultProfile.MatchString(line) {
			inDefaultProfile = true
		}

		// We are inside the correct block, get the value of Default
		if inDefaultProfile && strings.HasPrefix(line, "Default=") {
			profileName := strings.TrimPrefix(line, "Default=")
			return profileName, nil
		}

		if inDefaultProfile && line == "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning profiles.ini: %s", err)
	}

	return "", fmt.Errorf("could not find default-release profile")
}

// Copies src to dst
func copyFile(src, dst string) error {
	srcFh, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source file: %s", err)
	}
	defer srcFh.Close()

	dstFh, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create destination file: %s", err)
	}
	defer dstFh.Close()

	if _, err := io.Copy(dstFh, srcFh); err != nil {
		return fmt.Errorf("could not copy file: %s", err)
	}

	return nil
}

// Makes sure the query is a glob pattern
func convertToGlobPattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)

	// If pattern does not contain any wildcards (is no glob), make it a glob
	if !strings.ContainsAny(pattern, "*?[]") {
		return "*" + pattern + "*"
	}

	return pattern
}
