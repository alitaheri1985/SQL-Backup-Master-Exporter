package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const logDir = `C:\ProgramData\Key Metric Software\SQL Backup Master\logs`

var (
	backupSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sql_backup_status_succeeded",
		Help: "SQL Backup job success status",
	})
	backupFailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sql_backup_status_failed",
		Help: "SQL Backup job failure status",
	})
)

func getLatestLogFile() (string, error) {
	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		return "", err
	}

	var latestFile string
	var latestModTime time.Time

	fmt.Println("Found files in the directory:")
	for _, file := range files {
		fmt.Println("File:", file.Name())
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".xml") {
			continue
		}
		filePath := filepath.Join(logDir, file.Name())
		if file.ModTime().After(latestModTime) {
			latestModTime = file.ModTime()
			latestFile = filePath
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("no XML log file found")
	}

	return latestFile, nil
}

func getLastBackupStatus() string {
	logFile, err := getLatestLogFile()
	if err != nil {
		log.Printf("Error finding log file: %v", err)
		return "unknown"
	}

	data, err := ioutil.ReadFile(logFile)
	if err != nil {
		log.Printf("Error reading log file: %v", err)
		return "unknown"
	}

	lines := strings.Split(string(data), "\n")
	var lastStatus string

	// Look for success or failure lines in the XML file content
	for _, line := range lines {
		if strings.Contains(line, "Backup job succeeded") {
			lastStatus = "succeeded"
		} else if strings.Contains(line, "Backup job failed") {
			lastStatus = "failed"
		}
	}

	log.Printf("Last backup status: %s", lastStatus)
	return lastStatus
}

func updateMetrics() {
	status := getLastBackupStatus()
	backupSuccess.Set(0)
	backupFailed.Set(0)

	if status == "succeeded" {
		backupSuccess.Set(1)
	} else if status == "failed" {
		backupFailed.Set(1)
	}
}

func main() {
	prometheus.MustRegister(backupSuccess)
	prometheus.MustRegister(backupFailed)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		for {
			updateMetrics()
			time.Sleep(10 * time.Second)
		}
	}()

	log.Println("Prometheus exporter running on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
