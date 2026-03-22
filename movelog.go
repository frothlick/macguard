package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type MovementRecord struct {
	Time     string  `json:"t"`
	AvgMag   float64 `json:"avg"`
	PeakMag  float64 `json:"peak"`
	Lid      *bool   `json:"lid,omitempty"`
	LidAngle float64 `json:"lidAngle"`
	Tilt     float64 `json:"tilt"`
	Zone     string  `json:"zone"`
}

type DayLog struct {
	Date    string           `json:"date"`
	Records []MovementRecord `json:"records"`
}

type TrainingRecord struct {
	Time     string  `json:"t"`
	AvgMag   float64 `json:"avg"`
	PeakMag  float64 `json:"peak"`
	Lid      *bool   `json:"lid,omitempty"`
	LidAngle float64 `json:"lidAngle"`
	Tilt     float64 `json:"tilt"`
	Zone     string  `json:"zone"`
}

type TrainingSession struct {
	Start   string           `json:"start"`
	Records []TrainingRecord `json:"records"`
}

func isLidOpen() bool {
	out, err := exec.Command("ioreg", "-r", "-k", "AppleClamshellState", "-d", "1").Output()
	if err != nil {
		return true // assume open on error
	}
	return !strings.Contains(string(out), `"AppleClamshellState" = Yes`)
}

func getLidAngle() float64 {
	bin := filepath.Join(filepath.Dir(os.Args[0]), "lidangle")
	if _, err := os.Stat(bin); err != nil {
		bin = "/Users/alexander.wipf/macguard/helpers/lidangle"
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		return -1
	}
	var result struct {
		Status string  `json:"status"`
		Angle  float64 `json:"angle"`
	}
	if json.Unmarshal(out, &result) != nil || result.Status != "ok" {
		return -1
	}
	return result.Angle
}

func appendTrainingRecord(guard *GuardState) {
	guard.mu.Lock()
	if !guard.Training || guard.TrainingFile == "" {
		guard.mu.Unlock()
		return
	}
	avg := 0.0
	if guard.SecSamples > 0 {
		avg = guard.SecSum / float64(guard.SecSamples)
	}
	peak := guard.SecPeak
	tilt := 0.0
	if guard.SecTiltN > 0 {
		tilt = guard.SecTiltSum / float64(guard.SecTiltN)
	}
	guard.SecPeak = 0
	guard.SecSum = 0
	guard.SecSamples = 0
	guard.SecTiltSum = 0
	guard.SecTiltN = 0
	guard.SecondStart = time.Now().Truncate(time.Second)
	filePath := guard.TrainingFile
	guard.mu.Unlock()

	lid := isLidOpen()
	angle := getLidAngle()
	rec := TrainingRecord{
		Time:     time.Now().Format("15:04:05"),
		AvgMag:   avg,
		PeakMag:  peak,
		Lid:      &lid,
		LidAngle: angle,
		Tilt:     math.Round(tilt*10) / 10,
		Zone:     classifyMovementFull(avg, peak, tilt),
	}

	var session TrainingSession
	if data, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(data, &session)
	}
	session.Records = append(session.Records, rec)

	data, err := json.Marshal(session)
	if err != nil {
		return
	}
	tmp := filePath + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, filePath)
}

func moveLogDir() string {
	dir := "/Users/alexander.wipf/.macguard"
	os.MkdirAll(dir, 0755)
	return dir
}

func moveLogPath(date string) string {
	return filepath.Join(moveLogDir(), date+".json")
}

func classifyMovement(avg float64) string {
	return classifyMovementFull(avg, 0, 0)
}

func classifyMovementFull(avg, peak, tilt float64) string {
	switch {
	case avg >= 0.040:
		return "impact"
	case avg >= 0.015:
		return "motion"
	case avg >= 0.004:
		return "lap"
	case tilt > 15 && peak >= 0.01:
		return "lap"
	case avg >= 0.001:
		return "desk"
	case peak >= 0.01:
		return "desk"
	default:
		return "resting"
	}
}

type LocationRecord struct {
	Time    string  `json:"time"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	City    string  `json:"city,omitempty"`
	Region  string  `json:"region,omitempty"`
	Country string  `json:"country,omitempty"`
	Source  string  `json:"source"`
	Precise bool    `json:"precise"`
}

func locationLogPath() string {
	return filepath.Join(moveLogDir(), "locations.json")
}

func appendLocationRecord(geo *GeoLocation) {
	if geo == nil {
		return
	}
	rec := LocationRecord{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Lat:     geo.Lat,
		Lon:     geo.Lon,
		City:    geo.City,
		Region:  geo.Region,
		Country: geo.Country,
		Source:  geo.ISP,
		Precise: geo.Precise,
	}

	path := locationLogPath()
	var records []LocationRecord
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &records)
	}

	// Skip if same position as last record (within ~100m)
	if len(records) > 0 {
		last := records[len(records)-1]
		if abs(last.Lat-rec.Lat) < 0.001 && abs(last.Lon-rec.Lon) < 0.001 {
			return
		}
	}

	records = append(records, rec)
	// Keep last 500 entries
	if len(records) > 500 {
		records = records[len(records)-500:]
	}

	data, err := json.Marshal(records)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, path)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func appendMovementRecord(guard *GuardState) {
	guard.mu.Lock()
	avg := 0.0
	if guard.MagSamples > 0 {
		avg = guard.MagSum / float64(guard.MagSamples)
	}
	peak := guard.MagPeak
	tilt := 0.0
	if guard.TiltSamples > 0 {
		tilt = guard.TiltSum / float64(guard.TiltSamples)
	}
	guard.MagPeak = 0
	guard.MagSum = 0
	guard.MagSamples = 0
	guard.TiltSum = 0
	guard.TiltSamples = 0
	guard.MinuteStart = time.Now().Truncate(time.Minute)
	guard.mu.Unlock()

	lid := isLidOpen()
	angle := getLidAngle()
	now := time.Now()
	date := now.Format("2006-01-02")
	rec := MovementRecord{
		Time:     now.Format("15:04"),
		AvgMag:   avg,
		PeakMag:  peak,
		Lid:      &lid,
		LidAngle: angle,
		Tilt:     math.Round(tilt*10) / 10,
		Zone:     classifyMovementFull(avg, peak, tilt),
	}

	path := moveLogPath(date)
	var day DayLog

	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &day)
	}
	day.Date = date
	// Clean up legacy records from before lid tracking was added
	for i := range day.Records {
		if day.Records[i].Lid != nil && !*day.Records[i].Lid && day.Records[i].LidAngle == 0 && day.Records[i].Tilt == 0 {
			day.Records[i].Lid = nil
		}
	}
	day.Records = append(day.Records, rec)

	data, err := json.Marshal(day)
	if err != nil {
		fmt.Fprintf(os.Stderr, "movelog marshal: %v\n", err)
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "movelog write: %v\n", err)
		return
	}
	os.Rename(tmp, path)
}
