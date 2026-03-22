package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

type GuardState struct {
	mu sync.Mutex

	Armed     bool
	Mode      string // "move" or "geo"
	Moving    bool
	MagEWMA   float64
	LastAlert time.Time
	ArmedAt   time.Time
	ArmDelay  int // countdown seconds remaining (0 = fully armed)

	// Geo-fence
	AnchorLat float64
	AnchorLon float64

	// Calibration
	MagBaseline   float64 // noise floor to subtract
	TiltBaseline  float64 // tilt offset on flat surface
	Calibrating   bool
	CalibSamples  int
	CalibSum      float64
	CalibTiltSum  float64

	// Movement tracking
	MagPeak     float64   // peak magnitude this minute
	MagSum      float64   // sum of magnitudes this minute
	MagSamples  int       // sample count this minute
	MinuteStart time.Time // start of current minute

	// Tilt tracking (base angle from horizontal)
	TiltSum     float64
	TiltSamples int
	SecTiltSum  float64
	SecTiltN    int

	// Per-second training
	SecPeak     float64
	SecSum      float64
	SecSamples  int
	SecondStart time.Time
	Training    bool
	TrainingFile string

	// Config
	Threshold float64
	Cooldown  time.Duration
	Token     string
	ChatID    int64

	// Notifications
	SMTPHost    string
	SMTPUser    string
	SMTPPass    string
	NotifyEmail string

	// User settings (persisted)
	DefaultDelay    int  // default arm delay in seconds
	NotifyTelegram  bool // send alerts via Telegram
	NotifyEmailFlag bool // send alerts via Email
}

type UserSettings struct {
	DefaultDelay   int     `json:"defaultDelay"`
	NotifyTelegram bool    `json:"notifyTelegram"`
	NotifyEmail    bool    `json:"notifyEmail"`
	EmailAddress   string  `json:"emailAddress"`
	SMTPHost       string  `json:"smtpHost"`
	SMTPUser       string  `json:"smtpUser"`
	SMTPPass       string  `json:"smtpPass"`
	Baseline       float64 `json:"baseline"`
	TiltBaseline   float64 `json:"tiltBaseline"`
	TelegramChatID int64   `json:"telegramChatId,omitempty"`
}

func settingsPath() string {
	return filepath.Join(moveLogDir(), "settings.json")
}

func loadSettings(guard *GuardState) {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		// Defaults
		guard.NotifyTelegram = true
		guard.NotifyEmailFlag = guard.SMTPHost != "" && guard.NotifyEmail != ""
		return
	}
	var s UserSettings
	if json.Unmarshal(data, &s) == nil {
		guard.DefaultDelay = s.DefaultDelay
		guard.NotifyTelegram = s.NotifyTelegram
		guard.NotifyEmailFlag = s.NotifyEmail
		if s.EmailAddress != "" {
			guard.NotifyEmail = s.EmailAddress
		}
		guard.MagBaseline = s.Baseline
		guard.TiltBaseline = s.TiltBaseline
		if s.TelegramChatID != 0 {
			guard.ChatID = s.TelegramChatID
		}
		if s.SMTPHost != "" {
			guard.SMTPHost = s.SMTPHost
		}
		if s.SMTPUser != "" {
			guard.SMTPUser = s.SMTPUser
		}
		if s.SMTPPass != "" {
			guard.SMTPPass = s.SMTPPass
		}
	}
}

func saveSettings(guard *GuardState) {
	guard.mu.Lock()
	s := UserSettings{
		DefaultDelay:   guard.DefaultDelay,
		NotifyTelegram: guard.NotifyTelegram,
		NotifyEmail:    guard.NotifyEmailFlag,
		EmailAddress:   guard.NotifyEmail,
		SMTPHost:       guard.SMTPHost,
		SMTPUser:       guard.SMTPUser,
		SMTPPass:       guard.SMTPPass,
		Baseline:       guard.MagBaseline,
		TiltBaseline:   guard.TiltBaseline,
		TelegramChatID: guard.ChatID,
	}
	guard.mu.Unlock()
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	tmp := settingsPath() + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, settingsPath())
}

type StatusResponse struct {
	Status      string       `json:"status"`
	Mode        string       `json:"mode,omitempty"`
	Moving      bool         `json:"moving,omitempty"`
	Magnitude   float64      `json:"magnitude,omitempty"`
	Baseline    float64      `json:"baseline"`
	Calibrating bool         `json:"calibrating,omitempty"`
	LastAlert   string       `json:"lastAlert,omitempty"`
	ArmedAt     string       `json:"armedAt,omitempty"`
	ArmDelay    int          `json:"armDelay,omitempty"`
	Notify      NotifyStatus `json:"notify"`
}

type NotifyStatus struct {
	Telegram bool `json:"telegram"`
	Email    bool `json:"email"`
}

func main() {
	port := flag.Int("port", 8421, "HTTP control port")
	sensitivity := flag.Float64("sensitivity", 0.045, "EWMA threshold for movement alert")
	cooldown := flag.Duration("cooldown", 60*time.Second, "Min time between alerts")
	flag.Parse()

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "macguard requires root privileges for accelerometer access.")
		fmt.Fprintln(os.Stderr, "Run with: sudo -E macguard")
		os.Exit(1)
	}

	const defaultBotToken = "8723096596:AAEWWdqZwV-c5Wxww0DoKwHT4XN_VCkTttE"
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		token = defaultBotToken
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	var chatID int64
	if chatIDStr != "" {
		fmt.Sscanf(chatIDStr, "%d", &chatID)
	}

	guard := &GuardState{
		Threshold:   *sensitivity,
		Cooldown:    *cooldown,
		Token:       token,
		ChatID:      chatID,
		MinuteStart: time.Now().Truncate(time.Minute),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		NotifyEmail: os.Getenv("NOTIFY_EMAIL"),
	}
	loadSettings(guard)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create shared memory ring buffer for accelerometer
	accelRing, err := shm.CreateRing(shm.NameAccel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating accel shm: %v\n", err)
		os.Exit(1)
	}
	defer accelRing.Close()
	defer accelRing.Unlink()

	// Start sensor worker
	sensorReady := make(chan struct{})
	sensorErr := make(chan error, 1)
	go func() {
		close(sensorReady)
		if err := sensor.Run(sensor.Config{
			AccelRing: accelRing,
			Restarts:  0,
		}); err != nil {
			sensorErr <- err
		}
	}()

	select {
	case <-sensorReady:
	case err := <-sensorErr:
		fmt.Fprintf(os.Stderr, "sensor failed: %v\n", err)
		os.Exit(1)
	case <-ctx.Done():
		return
	}

	time.Sleep(100 * time.Millisecond)

	// Start sensor monitoring loop
	go monitorLoop(ctx, guard, accelRing)

	// Telegram bot command handler
	go telegramBotHandler(ctx, guard)

	// Start HTTP control API
	startHTTPServer(guard, *port)

	fmt.Fprintf(os.Stderr, "macguard: listening on :%d (disarmed)\n", *port)
	fmt.Fprintf(os.Stderr, "macguard: arm via POST /arm, disarm via POST /disarm\n")

	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "\nmacguard: shutting down")
}

func telegramBotHandler(ctx context.Context, guard *GuardState) {
	token := guard.Token
	if token == "" {
		return
	}
	client := &http.Client{Timeout: 35 * time.Second}
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		u := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30&allowed_updates=[\"message\"]", token, offset)
		resp, err := client.Get(u)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		var result struct {
			OK     bool `json:"ok"`
			Result []struct {
				UpdateID int64 `json:"update_id"`
				Message  struct {
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"result"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if !result.OK {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, upd := range result.Result {
			offset = upd.UpdateID + 1
			chatID := upd.Message.Chat.ID
			cmd := upd.Message.Text

			// Only respond to the configured chat ID (except /start)
			guard.mu.Lock()
			ownerChat := guard.ChatID
			guard.mu.Unlock()

			cmdBase := cmd
			if idx := strings.Index(cmd, " "); idx > 0 {
				cmdBase = cmd[:idx]
			}

			switch cmdBase {
			case "/start":
				text := "Welcome to MacGuard. Enter the following ID in your MacGuard Settings under Telegram Chat ID."
				sendTelegramMessage(token, chatID, text)
				sendTelegramMessage(token, chatID, fmt.Sprintf("%d", chatID))

			case "/help", "/?":
				helpText := "*MacGuard Commands*\n\n" +
					"/arm — Arm (movement mode)\n" +
					"/arm\\_geo — Arm (geo-fence mode)\n" +
					"/disarm — Disarm\n" +
					"/status — Show guard status\n" +
					"/location — Send current location\n" +
					"/msg — Display message on Mac\n" +
					"/msg _text_ — Display custom message\n" +
					"/help — Show this help"
				sendTelegramMessage(token, chatID, helpText)

			case "/arm":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				guard.mu.Lock()
				guard.Armed = true
				guard.Mode = "move"
				guard.Moving = false
				guard.MagEWMA = 0
				guard.ArmedAt = time.Now()
				guard.mu.Unlock()
				sendAlert(guard, token, chatID, 0)

			case "/arm_geo":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				geo := getLocation()
				if geo == nil || !geo.Precise {
					sendTelegramMessage(token, chatID, "Geo-fence requires precise location. Not available right now.")
					continue
				}
				guard.mu.Lock()
				guard.Armed = true
				guard.Mode = "geo"
				guard.Moving = false
				guard.MagEWMA = 0
				guard.ArmedAt = time.Now()
				guard.AnchorLat = geo.Lat
				guard.AnchorLon = geo.Lon
				guard.mu.Unlock()
				sendAlert(guard, token, chatID, 0)

			case "/disarm":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				guard.mu.Lock()
				guard.Armed = false
				guard.Moving = false
				guard.MagEWMA = 0
				guard.mu.Unlock()
				sendTelegramMessage(token, chatID, "Disarmed.")

			case "/status":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				guard.mu.Lock()
				armed := guard.Armed
				mode := guard.Mode
				moving := guard.Moving
				mag := guard.MagEWMA
				guard.mu.Unlock()
				status := "Disarmed"
				if armed {
					status = fmt.Sprintf("Armed (%s)", mode)
				}
				text := fmt.Sprintf("*MacGuard Status*\n%s\nMoving: %v\nMagnitude: `%.3fg`", status, moving, mag)
				sendTelegramMessage(token, chatID, text)

			case "/msg":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				msgText := "This Mac is protected by MacGuard. The owner has been notified of your activity."
				if len(cmd) > 5 {
					msgText = strings.TrimSpace(cmd[4:])
				}
				escaped := strings.ReplaceAll(msgText, `"`, `\"`)
				escaped = strings.ReplaceAll(escaped, `'`, `'"'"'`)
				displayCmd := exec.Command("su", "-", "alexander.wipf", "-c",
					fmt.Sprintf(`osascript -e 'display dialog "%s" with title "MacGuard" buttons {"OK"} default button "OK" with icon caution'`,
						escaped))
				displayCmd.Run()
				sendTelegramMessage(token, chatID, "Message displayed.")

			case "/location":
				if chatID != ownerChat {
					sendTelegramMessage(token, chatID, "Not authorized.")
					continue
				}
				geo := getLocation()
				if geo == nil {
					sendTelegramMessage(token, chatID, "Location unavailable.")
					continue
				}
				locType := "Approximate"
				if geo.Precise {
					locType = "Precise"
				}
				text := fmt.Sprintf("*%s Location*\n%s, %s, %s\n_via %s_", locType, geo.City, geo.Region, geo.Country, geo.ISP)
				sendTelegramMessage(token, chatID, text)
				sendTelegramLocation(token, chatID, geo.Lat, geo.Lon)
			}
		}
	}
}

func monitorLoop(ctx context.Context, guard *GuardState, ring *shm.RingBuffer) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var lastTotal uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		samples, newTotal := ring.ReadNew(lastTotal, shm.AccelScale)
		lastTotal = newTotal
		if len(samples) > 200 {
			samples = samples[len(samples)-200:]
		}

		guard.mu.Lock()
		for _, s := range samples {
			mag := math.Sqrt(s.X*s.X+s.Y*s.Y+s.Z*s.Z) - 1.0
			if mag < 0 {
				mag = 0
			}

			// Calibration sampling
			if guard.Calibrating {
				guard.CalibSum += mag
				calibTilt := math.Atan2(math.Sqrt(s.X*s.X+s.Y*s.Y), math.Abs(s.Z)) * 180 / math.Pi
				guard.CalibTiltSum += calibTilt
				guard.CalibSamples++
				continue
			}

			// Subtract noise floor
			mag -= guard.MagBaseline
			if mag < 0 {
				mag = 0
			}

			guard.MagEWMA = 0.05*mag + 0.95*guard.MagEWMA

			// Movement tracking
			guard.MagSum += mag
			guard.MagSamples++
			if mag > guard.MagPeak {
				guard.MagPeak = mag
			}

			// Tilt from gravity vector (degrees from horizontal, baseline-corrected)
			tilt := math.Atan2(math.Sqrt(s.X*s.X+s.Y*s.Y), math.Abs(s.Z))*180/math.Pi - guard.TiltBaseline
			guard.TiltSum += tilt
			guard.TiltSamples++
			guard.SecTiltSum += tilt
			guard.SecTiltN++

			// Per-second training tracking
			guard.SecSum += mag
			guard.SecSamples++
			if mag > guard.SecPeak {
				guard.SecPeak = mag
			}
		}

		// Flush training log every second
		now := time.Now()
		if guard.Training && guard.SecSamples > 0 && now.Truncate(time.Second).After(guard.SecondStart) {
			guard.mu.Unlock()
			appendTrainingRecord(guard)
			guard.mu.Lock()
		}

		// Flush movement log every minute
		if guard.MagSamples > 0 && now.Truncate(time.Minute).After(guard.MinuteStart) {
			guard.mu.Unlock()
			appendMovementRecord(guard)
			guard.mu.Lock()
		}

		if guard.Armed && guard.ArmDelay <= 0 {
			if guard.MagEWMA >= guard.Threshold {
				if !guard.Moving {
					guard.Moving = true
				}
				// Send alert if cooldown has elapsed
				if time.Since(guard.LastAlert) >= guard.Cooldown {
					mag := guard.MagEWMA
					token := guard.Token
					chatID := guard.ChatID
					mode := guard.Mode
					guard.LastAlert = time.Now()

					if mode == "geo" {
						// Geo-fence: check location on movement
						anchorLat := guard.AnchorLat
						anchorLon := guard.AnchorLon
						guard.mu.Unlock()
						go checkGeoFence(guard, token, chatID, anchorLat, anchorLon, mag)
					} else {
						guard.mu.Unlock()
						go sendAlert(guard, token, chatID, mag)
					}
					continue
				}
			} else if guard.MagEWMA < 0.020 {
				// Hysteresis: only clear moving state when well below threshold
				guard.Moving = false
			}
		}
		guard.mu.Unlock()
	}
}

func startHTTPServer(guard *GuardState, port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /arm", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Mode  string `json:"mode"`
			Delay int    `json:"delay"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Mode == "" {
			body.Mode = "move"
		}
		if body.Mode != "move" && body.Mode != "geo" {
			http.Error(w, "mode must be move or geo", 400)
			return
		}

		if body.Mode == "geo" {
			// Need precise location for geo-fence
			geo := getLocation()
			if geo == nil || !geo.Precise {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"status": "error",
					"error":  "geo mode requires precise location signal",
				})
				return
			}
			guard.mu.Lock()
			guard.AnchorLat = geo.Lat
			guard.AnchorLon = geo.Lon
			guard.mu.Unlock()
		}

		if body.Delay > 0 {
			guard.mu.Lock()
			guard.Armed = true
			guard.Mode = body.Mode
			guard.Moving = false
			guard.MagEWMA = 0
			guard.ArmDelay = body.Delay
			guard.ArmedAt = time.Now().Add(time.Duration(body.Delay) * time.Second)
			guard.mu.Unlock()

			// Countdown in background
			go func() {
				for i := body.Delay; i > 0; i-- {
					time.Sleep(1 * time.Second)
					guard.mu.Lock()
					guard.ArmDelay = i - 1
					guard.mu.Unlock()
				}
				guard.mu.Lock()
				guard.ArmedAt = time.Now()
				guard.mu.Unlock()
				fmt.Fprintf(os.Stderr, "macguard: ARMED (%s)\n", body.Mode)
				go sendAlert(guard, guard.Token, guard.ChatID, 0)
			}()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status": "arming",
				"mode":   body.Mode,
				"delay":  body.Delay,
			})
			return
		}

		guard.mu.Lock()
		guard.Armed = true
		guard.Mode = body.Mode
		guard.Moving = false
		guard.MagEWMA = 0
		guard.ArmDelay = 0
		guard.ArmedAt = time.Now()
		guard.mu.Unlock()

		fmt.Fprintf(os.Stderr, "macguard: ARMED (%s)\n", body.Mode)
		go sendTelegramMessage(guard.Token, guard.ChatID, fmt.Sprintf("*macguard armed* (%s mode)", body.Mode))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "armed", "mode": body.Mode})
	})

	mux.HandleFunc("POST /disarm", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		guard.Armed = false
		guard.Moving = false
		guard.ArmDelay = 0
		guard.Mode = ""
		guard.mu.Unlock()

		fmt.Fprintf(os.Stderr, "macguard: DISARMED\n")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "disarmed"})
	})

	mux.HandleFunc("POST /calibrate", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		if guard.Calibrating {
			guard.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "already calibrating"})
			return
		}
		guard.Calibrating = true
		guard.CalibSum = 0
		guard.CalibTiltSum = 0
		guard.CalibSamples = 0
		guard.mu.Unlock()

		// Sample for 3 seconds
		go func() {
			time.Sleep(3 * time.Second)
			guard.mu.Lock()
			if guard.CalibSamples > 0 {
				guard.MagBaseline = guard.CalibSum / float64(guard.CalibSamples)
				guard.TiltBaseline = guard.CalibTiltSum / float64(guard.CalibSamples)
			}
			guard.Calibrating = false
			guard.MagEWMA = 0
			baseline := guard.MagBaseline
			tiltBase := guard.TiltBaseline
			guard.mu.Unlock()
			fmt.Fprintf(os.Stderr, "macguard: calibrated baseline=%.6fg tilt=%.1f° (%d samples)\n", baseline, tiltBase, guard.CalibSamples)
			saveSettings(guard)
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "calibrating"})
	})

	mux.HandleFunc("GET /settings", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		s := UserSettings{
			DefaultDelay:   guard.DefaultDelay,
			NotifyTelegram: guard.NotifyTelegram,
			NotifyEmail:    guard.NotifyEmailFlag,
			EmailAddress:   guard.NotifyEmail,
			SMTPHost:       guard.SMTPHost,
			SMTPUser:       guard.SMTPUser,
			SMTPPass:       guard.SMTPPass,
			Baseline:       guard.MagBaseline,
			TelegramChatID: guard.ChatID,
		}
		guard.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
	})

	mux.HandleFunc("POST /settings", func(w http.ResponseWriter, r *http.Request) {
		var s UserSettings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "invalid json", 400)
			return
		}
		guard.mu.Lock()
		guard.DefaultDelay = s.DefaultDelay
		guard.NotifyTelegram = s.NotifyTelegram
		guard.NotifyEmailFlag = s.NotifyEmail
		if s.EmailAddress != "" {
			guard.NotifyEmail = s.EmailAddress
		}
		if s.TelegramChatID != 0 {
			guard.ChatID = s.TelegramChatID
		}
		guard.SMTPHost = s.SMTPHost
		guard.SMTPUser = s.SMTPUser
		if s.SMTPPass != "" {
			guard.SMTPPass = s.SMTPPass
		}
		guard.mu.Unlock()
		saveSettings(guard)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
	})

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		resp := StatusResponse{
			Magnitude:   guard.MagEWMA,
			Baseline:    guard.MagBaseline,
			Calibrating: guard.Calibrating,
			Notify: NotifyStatus{
				Telegram: guard.NotifyTelegram && guard.Token != "",
				Email:    guard.NotifyEmailFlag && guard.SMTPHost != "" && guard.NotifyEmail != "",
			},
		}
		if guard.Armed {
			if guard.ArmDelay > 0 {
				resp.Status = "arming"
			} else {
				resp.Status = "armed"
			}
			resp.Mode = guard.Mode
			resp.Moving = guard.Moving
			resp.ArmDelay = guard.ArmDelay
			resp.ArmedAt = guard.ArmedAt.Format(time.RFC3339)
			if !guard.LastAlert.IsZero() {
				resp.LastAlert = guard.LastAlert.Format(time.RFC3339)
			}
		} else {
			resp.Status = "disarmed"
		}
		guard.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /location", func(w http.ResponseWriter, r *http.Request) {
		geo := getLocation()
		w.Header().Set("Content-Type", "application/json")
		if geo == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
			return
		}
		go appendLocationRecord(geo)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"precise": geo.Precise,
			"city":    geo.City,
			"region":  geo.Region,
			"country": geo.Country,
			"isp":     geo.ISP,
			"lat":     geo.Lat,
			"lon":     geo.Lon,
		})
	})

	mux.HandleFunc("GET /locations", func(w http.ResponseWriter, r *http.Request) {
		path := locationLogPath()
		data, err := os.ReadFile(path)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("GET /activity/days", func(w http.ResponseWriter, r *http.Request) {
		dir := moveLogDir()
		entries, _ := os.ReadDir(dir)
		var days []string
		for _, e := range entries {
			name := e.Name()
			if filepath.Ext(name) == ".json" && len(name) == len("2006-01-02.json") && name[4] == '-' {
				days = append(days, name[:len(name)-5])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(days)
	})

	mux.HandleFunc("POST /training/start", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		if guard.Training {
			guard.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "already_recording", "file": filepath.Base(guard.TrainingFile)})
			return
		}
		fname := fmt.Sprintf("training-%s.json", time.Now().Format("20060102-150405"))
		fpath := filepath.Join(moveLogDir(), fname)
		session := TrainingSession{Start: time.Now().Format("2006-01-02 15:04:05")}
		data, _ := json.Marshal(session)
		os.WriteFile(fpath, data, 0644)
		guard.Training = true
		guard.TrainingFile = fpath
		guard.SecSum = 0
		guard.SecSamples = 0
		guard.SecPeak = 0
		guard.SecondStart = time.Now().Truncate(time.Second)
		guard.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "recording", "file": fname})
	})

	mux.HandleFunc("POST /training/stop", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		if !guard.Training {
			guard.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "not_recording"})
			return
		}
		guard.Training = false
		fpath := guard.TrainingFile
		guard.TrainingFile = ""
		guard.mu.Unlock()

		count := 0
		if data, err := os.ReadFile(fpath); err == nil {
			var session TrainingSession
			if json.Unmarshal(data, &session) == nil {
				count = len(session.Records)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "stopped", "file": filepath.Base(fpath), "records": count})
	})

	mux.HandleFunc("GET /training/status", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		training := guard.Training
		fname := ""
		if guard.TrainingFile != "" {
			fname = filepath.Base(guard.TrainingFile)
		}
		guard.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"recording": training, "file": fname})
	})

	mux.HandleFunc("GET /training/data", func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		if file == "" || strings.Contains(file, "/") || strings.Contains(file, "..") {
			http.Error(w, "invalid file", 400)
			return
		}
		data, err := os.ReadFile(filepath.Join(moveLogDir(), file))
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(dashboardHTML))
	})

	mux.HandleFunc("POST /message", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
			http.Error(w, "need {\"text\":\"...\"}", 400)
			return
		}
		go exec.Command("osascript", "-e",
			fmt.Sprintf(`display dialog %q with title "macguard" buttons {"OK"} default button "OK"`, body.Text),
		).Run()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	})

	mux.HandleFunc("GET /activity/range", func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			http.Error(w, "need from and to", 400)
			return
		}
		fromDate, err1 := time.Parse("2006-01-02", from)
		toDate, err2 := time.Parse("2006-01-02", to)
		if err1 != nil || err2 != nil {
			http.Error(w, "invalid date format", 400)
			return
		}
		var allDays []DayLog
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			path := moveLogPath(dateStr)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var day DayLog
			if json.Unmarshal(data, &day) == nil {
				allDays = append(allDays, day)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allDays)
	})

	mux.HandleFunc("GET /activity", func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		path := moveLogPath(date)
		data, err := os.ReadFile(path)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "no data", "date": date})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("POST /capture", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Duration int `json:"duration"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Duration <= 0 {
			body.Duration = 10
		}
		if body.Duration > 60 {
			body.Duration = 60
		}

		// Run capture in the GUI user's session (uid 501 = first user)
		dir := filepath.Dir(os.Args[0])
		capturePath := filepath.Join(dir, "capture")

		cmd := exec.Command("launchctl", "asuser", "501",
			capturePath, fmt.Sprintf("%d", body.Duration))
		out, err := cmd.Output()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		// Pass through the JSON from capture tool
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
