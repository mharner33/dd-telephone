package hosts

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

var hostsList = []string{"tele0", "tele1", "tele2", "tele3", "tele4"}
var hostMsgMap = map[string]string{
	"tele0": "http://tele0:8080/api/v1/message",
	"tele1": "http://tele1:8081/api/v1/message",
	"tele2": "http://tele2:8082/api/v1/message",
	"tele3": "http://tele3:8083/api/v1/message",
	"tele4": "http://tele4:8084/api/v1/message",
}

// Function variables for mocking in tests
var getHostnameFunc = os.Hostname
var checkHostHealthFunc = CheckHostHealth
var hostHealthMap = map[string]string{
	"tele0": "http://tele0:8080/api/v1/health",
	"tele1": "http://tele1:8081/api/v1/health",
	"tele2": "http://tele2:8082/api/v1/health",
	"tele3": "http://tele3:8083/api/v1/health",
	"tele4": "http://tele4:8084/api/v1/health",
}

// GetNextHost returns the next healthy host in the rotation
func GetNextHost() string {
	hostname, err := getHostnameFunc()
	if err != nil {
		logrus.WithError(err).Error("Error getting hostname")
		return ""
	}

	currentIndex := -1
	for i, host := range hostsList {
		if host == hostname {
			currentIndex = i
			break
		}
	}

	// If current hostname not found in array, start from 0
	if currentIndex == -1 {
		logrus.Info("Hostname not found in array, starting from 0")
		currentIndex = -1

	}

	// Try each host starting from the next one
	for i := 1; i <= len(hostsList); i++ {
		nextIndex := (currentIndex + i) % len(hostsList)
		nextHost := hostsList[nextIndex]

		// Check health of this host
		if checkHostHealthFunc(nextHost) {
			return nextHost
		}
	}

	// If no healthy host found, return the immediate next host anyway
	nextIndex := (currentIndex + 1) % len(hostsList)
	return hostsList[nextIndex]
}

// CheckHostHealth checks if a given host is healthy
func CheckHostHealth(host string) bool {
	healthURL, exists := hostHealthMap[host]
	if !exists {
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetNextHostURL returns the message URL for the next host in the rotation
func GetNextHostURL() string {
	nextHost := GetNextHost()

	// If next host is the first in the list, we've completed the cycle
	if nextHost == hostsList[0] {
		return ""
	}

	if url, exists := hostMsgMap[nextHost]; exists {
		return url
	}
	return ""
}

// GetNextHostHealth checks the health of the next host in the rotation
func GetNextHostHealth() bool {
	nextHost := GetNextHost()
	healthURL, exists := hostHealthMap[nextHost]
	if !exists {
		logrus.WithField("host", nextHost).Error("Health check failed: host not found in health map")
		return false
	}

	resp, err := http.Get(healthURL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"host":  nextHost,
			"error": err,
		}).Error("Health check failed")
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"host":        nextHost,
			"status_code": resp.StatusCode,
		}).Error("Health check failed")
		return false
	}

	return true
}
