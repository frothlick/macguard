package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/smtp"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GeoLocation holds location data.
type GeoLocation struct {
	City    string  `json:"city"`
	Region  string  `json:"regionName"`
	Country string  `json:"country"`
	ISP     string  `json:"isp"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Status  string  `json:"status"`
	Precise    bool   `json:"-"`
	VPN        bool   `json:"vpn,omitempty"`
	VPNCity    string `json:"-"`
	VPNRegion  string `json:"-"`
	VPNCountry string `json:"-"`
}

var (
	geoCache     *GeoLocation
	geoCacheTime time.Time
	geoMu        sync.Mutex
)

func fetchIPGeo() *GeoLocation {
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
	return &geo
}

// PreciseLocation from CoreLocation via the Swift helper.
type PreciseLocation struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Accuracy float64 `json:"acc"`
	Status   string  `json:"status"`
}

func getPreciseLocation() *PreciseLocation {
	locateBin := "/Users/alexander.wipf/macguard/Locate.app/Contents/MacOS/locate"

	// Run as the GUI user so CoreLocation inherits TCC permissions
	cmd := exec.Command("su", "-", "alexander.wipf", "-c", locateBin)
	out, err := cmd.Output()
	if err != nil {
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

// haversineDist returns distance in km between two lat/lon points.
func haversineDist(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// reverseGeocode uses Nominatim to get city/region/country from coordinates.
func reverseGeocode(lat, lon float64) (city, region, country string) {
	u := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%.6f&lon=%.6f&zoom=10", lat, lon)
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "macguard/0.2")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var result struct {
		Address struct {
			City    string `json:"city"`
			Town    string `json:"town"`
			Village string `json:"village"`
			State   string `json:"state"`
			Country string `json:"country"`
		} `json:"address"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return
	}
	city = result.Address.City
	if city == "" {
		city = result.Address.Town
	}
	if city == "" {
		city = result.Address.Village
	}
	region = result.Address.State
	country = result.Address.Country
	return
}

func getLocation() *GeoLocation {
	geoMu.Lock()
	defer geoMu.Unlock()

	// Return cached precise location for 5 min, cached IP location for 1 min
	if geoCache != nil {
		ttl := 1 * time.Minute
		if geoCache.ISP == "CoreLocation" {
			ttl = 5 * time.Minute
		}
		if time.Since(geoCacheTime) < ttl {
			return geoCache
		}
	}

	// Try precise CoreLocation first
	precise := getPreciseLocation()
	if precise != nil {
		// Get IP geolocation for VPN comparison
		ipGeo := fetchIPGeo()

		// Reverse geocode GPS coordinates for actual location name
		actualCity, actualRegion, actualCountry := reverseGeocode(precise.Lat, precise.Lon)
		if actualCity == "" {
			actualCity = fmt.Sprintf("%.4f, %.4f", precise.Lat, precise.Lon)
		}

		// Use IP geo for display name, detect VPN by distance
		vpn := false
		city := actualCity
		region := actualRegion
		country := actualCountry
		vpnCity := ""
		vpnRegion := ""
		vpnCountry := ""
		if ipGeo != nil {
			dist := haversineDist(precise.Lat, precise.Lon, ipGeo.Lat, ipGeo.Lon)
			if dist > 100 {
				vpn = true
				vpnCity = ipGeo.City
				vpnRegion = ipGeo.Region
				vpnCountry = ipGeo.Country
			}
		}

		geoCache = &GeoLocation{
			Lat:     precise.Lat,
			Lon:     precise.Lon,
			Status:  "success",
			City:    city,
			Region:  region,
			Country: country,
			ISP:     "CoreLocation",
			Precise: true,
			VPN:     vpn,
			VPNCity:    vpnCity,
			VPNRegion:  vpnRegion,
			VPNCountry: vpnCountry,
		}
		geoCacheTime = time.Now()
		return geoCache
	}

	// Fall back to IP geolocation
	geo := fetchIPGeo()
	if geo == nil {
		return nil
	}

	geoCache = geo
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

func sendAlert(guard *GuardState, token string, chatID int64, mag float64) {
	now := time.Now().Format("2006-01-02 15:04:05")
	var text string
	if mag == 0 {
		// Arm confirmation
		guard.mu.Lock()
		var modes []string
		if guard.LocalArmed { modes = append(modes, "local") }
		if guard.GeoArmed { modes = append(modes, "geo") }
		guard.mu.Unlock()
		modeStr := "unknown"
		if len(modes) > 0 { modeStr = strings.Join(modes, "+") }
		text = fmt.Sprintf("*macguard armed* (%s)\n_%s_", modeStr, now)
	} else {
		text = fmt.Sprintf("*Your Mac is being moved!*\n_%s_\nMagnitude: `%.3fg`", now, mag)
	}

	geo := getLocation()
	if geo != nil && mag > 0 {
		locType := "Approximate"
		if geo.Precise {
			locType = "Precise"
		}
		text += fmt.Sprintf("\n\n*%s Location*\n%s, %s, %s\n_via %s_",
			locType, geo.City, geo.Region, geo.Country, geo.ISP)
	}

	// Read notify preferences
	guard.mu.Lock()
	wantTelegram := guard.NotifyTelegram
	wantEmail := guard.NotifyEmailFlag
	smtpHost := guard.SMTPHost
	smtpUser := guard.SMTPUser
	smtpPass := guard.SMTPPass
	notifyEmail := guard.NotifyEmail
	guard.mu.Unlock()

	// Send via Telegram
	if wantTelegram && token != "" {
		if err := sendTelegramMessage(token, chatID, text); err != nil {
			fmt.Printf("telegram alert failed: %v\n", err)
		}
		if geo != nil && geo.Lat != 0 && mag > 0 {
			if err := sendTelegramLocation(token, chatID, geo.Lat, geo.Lon); err != nil {
				fmt.Printf("telegram location failed: %v\n", err)
			}
		}
	}

	// Send via Email
	if wantEmail && smtpHost != "" && notifyEmail != "" {
		sendEmailAlert(smtpHost, smtpUser, smtpPass, notifyEmail, text)
	}

	// Play alarm sound
	if mag > 0 {
		guard.mu.Lock()
		alarmEnabled := guard.AlarmEnabled
		alarmSound := guard.AlarmSound
		guard.mu.Unlock()
		if alarmEnabled {
			if alarmSound == "" {
				alarmSound = "Sosumi"
			}
			playAlarm(guard, alarmSound)
		}
	}
}

func sendEmailAlert(host, user, pass, to, body string) {
	subject := "macguard alert"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", user, to, subject, body)

	addr := host
	if !containsPort(host) {
		addr = host + ":465"
	}
	hostOnly := host
	if i := strings.LastIndex(host, ":"); i > 0 {
		hostOnly = host[:i]
	}

	// Port 465 = implicit TLS, port 587 = STARTTLS
	if strings.HasSuffix(addr, ":465") {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: hostOnly})
		if err != nil {
			fmt.Printf("email tls dial failed: %v\n", err)
			return
		}
		defer tlsConn.Close()
		c, err := smtp.NewClient(tlsConn, hostOnly)
		if err != nil {
			fmt.Printf("email client failed: %v\n", err)
			return
		}
		defer c.Close()
		auth := smtp.PlainAuth("", user, pass, hostOnly)
		if err := c.Auth(auth); err != nil {
			fmt.Printf("email auth failed: %v\n", err)
			return
		}
		if err := c.Mail(user); err != nil {
			fmt.Printf("email mail-from failed: %v\n", err)
			return
		}
		if err := c.Rcpt(to); err != nil {
			fmt.Printf("email rcpt failed: %v\n", err)
			return
		}
		w, err := c.Data()
		if err != nil {
			fmt.Printf("email data failed: %v\n", err)
			return
		}
		w.Write([]byte(msg))
		w.Close()
		c.Quit()
	} else {
		auth := smtp.PlainAuth("", user, pass, hostOnly)
		if err := smtp.SendMail(addr, auth, user, []string{to}, []byte(msg)); err != nil {
			fmt.Printf("email alert failed: %v\n", err)
		}
	}
}

func containsPort(host string) bool {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return true
		}
		if host[i] < '0' || host[i] > '9' {
			return false
		}
	}
	return false
}

// haversineDistance returns distance in meters between two lat/lon points.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth radius in meters
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func playAlarm(guard *GuardState, sound string) {
	if sound == "" {
		sound = "Sosumi"
	}
	stop := make(chan struct{})
	guard.mu.Lock()
	// Stop any existing alarm first
	if guard.alarmStop != nil {
		close(guard.alarmStop)
	}
	guard.alarmStop = stop
	guard.mu.Unlock()

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			cmd := exec.Command("launchctl", "asuser", "501", "afplay", "/System/Library/Sounds/"+sound+".aiff")
			cmd.Run()
			select {
			case <-stop:
				return
			case <-time.After(1500 * time.Millisecond):
			}
		}
	}()
}

func stopAlarm(guard *GuardState) {
	guard.mu.Lock()
	if guard.alarmStop != nil {
		close(guard.alarmStop)
		guard.alarmStop = nil
	}
	guard.mu.Unlock()
}

func checkGeoFence(guard *GuardState, token string, chatID int64, anchorLat, anchorLon, mag float64) {
	// Invalidate geo cache to force fresh GPS lookup
	geoMu.Lock()
	geoCacheTime = time.Time{}
	geoMu.Unlock()

	geo := getLocation()
	if geo == nil || !geo.Precise {
		// Can't verify location without precise signal, skip
		return
	}

	dist := haversineDistance(anchorLat, anchorLon, geo.Lat, geo.Lon)
	if dist < 50 {
		// Still within geo-fence radius, no alert
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	text := fmt.Sprintf("*Your Mac left the geo-fence!*\n_%s_\nDistance: `%.0fm` from anchor\nMagnitude: `%.3fg`",
		now, dist, mag)
	text += fmt.Sprintf("\n\n*Precise Location*\n%s, %s, %s", geo.City, geo.Region, geo.Country)

	if token != "" {
		if err := sendTelegramMessage(token, chatID, text); err != nil {
			fmt.Printf("telegram alert failed: %v\n", err)
		}
		if err := sendTelegramLocation(token, chatID, geo.Lat, geo.Lon); err != nil {
			fmt.Printf("telegram location failed: %v\n", err)
		}
	}

	guard.mu.Lock()
	smtpHost := guard.SMTPHost
	smtpUser := guard.SMTPUser
	smtpPass := guard.SMTPPass
	notifyEmail := guard.NotifyEmail
	alarmEnabled := guard.AlarmEnabled
	geoAlarmSound := guard.GeoAlarmSound
	guard.mu.Unlock()

	if smtpHost != "" && notifyEmail != "" {
		sendEmailAlert(smtpHost, smtpUser, smtpPass, notifyEmail, text)
	}

	// Play geo-fence alarm sound
	if alarmEnabled {
		if geoAlarmSound == "" {
			geoAlarmSound = "Submarine"
		}
		playAlarm(guard, geoAlarmSound)
	}
}
