package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// GeoLocation holds IP-based location data from ip-api.com.
type GeoLocation struct {
	City    string  `json:"city"`
	Region  string  `json:"regionName"`
	Country string  `json:"country"`
	ISP     string  `json:"isp"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Status  string  `json:"status"`
}

var (
	geoCache     *GeoLocation
	geoCacheTime time.Time
	geoMu        sync.Mutex
)

// PreciseLocation from CoreLocation via the Swift helper.
type PreciseLocation struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Accuracy float64 `json:"acc"`
	Status   string  `json:"status"`
}

func getPreciseLocation() *PreciseLocation {
	// Use the signed app bundle for CoreLocation permissions
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dir := filepath.Dir(exe)

	// Try app bundle first, then raw binary
	candidates := []string{
		filepath.Join(dir, "Locate.app", "Contents", "MacOS", "locate"),
		filepath.Join(dir, "locate"),
		"./Locate.app/Contents/MacOS/locate",
		"./locate",
	}

	var out []byte
	for _, path := range candidates {
		cmd := exec.Command(path)
		var cmdErr error
		out, cmdErr = cmd.Output()
		if cmdErr == nil {
			break
		}
	}
	if out == nil {
		return nil
	}

	var loc PreciseLocation
	if err := json.Unmarshal(out, &loc); err != nil {
		return nil
	}
	if loc.Status != "ok" {
		return nil
	}
	return &loc
}

func getLocation() *GeoLocation {
	geoMu.Lock()
	defer geoMu.Unlock()

	if geoCache != nil && time.Since(geoCacheTime) < 5*time.Minute {
		return geoCache
	}

	// Try precise CoreLocation first
	precise := getPreciseLocation()
	if precise != nil {
		geoCache = &GeoLocation{
			Lat:    precise.Lat,
			Lon:    precise.Lon,
			Status: "success",
			City:   fmt.Sprintf("Precise (%.0fm)", precise.Accuracy),
			ISP:    "CoreLocation",
		}
		geoCacheTime = time.Now()
		return geoCache
	}

	// Fall back to IP geolocation
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var geo GeoLocation
	if err := json.Unmarshal(body, &geo); err != nil {
		return nil
	}
	if geo.Status != "success" {
		return nil
	}

	geoCache = &geo
	geoCacheTime = time.Now()
	return geoCache
}

func sendTelegramMessage(token string, chatID int64, text string) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.PostForm(u, url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"Markdown"},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage: %s %s", resp.Status, body)
	}
	return nil
}

func sendTelegramLocation(token string, chatID int64, lat, lon float64) error {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendLocation", token)
	resp, err := http.PostForm(u, url.Values{
		"chat_id":   {strconv.FormatInt(chatID, 10)},
		"latitude":  {strconv.FormatFloat(lat, 'f', 6, 64)},
		"longitude": {strconv.FormatFloat(lon, 'f', 6, 64)},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendLocation: %s %s", resp.Status, body)
	}
	return nil
}

func sendMovementAlert(token string, chatID int64, mag float64) {
	now := time.Now().Format("2006-01-02 15:04:05")
	text := fmt.Sprintf("*Your Mac is being moved!*\n_%s_\nMagnitude: `%.3fg`", now, mag)

	geo := getLocation()
	if geo != nil {
		text += fmt.Sprintf("\n\n*Approximate Location*\n%s, %s, %s\n_via %s_",
			geo.City, geo.Region, geo.Country, geo.ISP)
	}

	if err := sendTelegramMessage(token, chatID, text); err != nil {
		fmt.Printf("alert send failed: %v\n", err)
	}

	if geo != nil && geo.Lat != 0 {
		if err := sendTelegramLocation(token, chatID, geo.Lat, geo.Lon); err != nil {
			fmt.Printf("location send failed: %v\n", err)
		}
	}
}
