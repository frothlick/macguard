package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

type GuardState struct {
	mu sync.Mutex

	Armed     bool
	Moving    bool
	MagEWMA   float64
	LastAlert time.Time
	ArmedAt   time.Time

	// Config
	Threshold float64
	Cooldown  time.Duration
	Token     string
	ChatID    int64
}

type StatusResponse struct {
	Status    string `json:"status"`
	Moving    bool   `json:"moving,omitempty"`
	Magnitude float64 `json:"magnitude,omitempty"`
	LastAlert string `json:"lastAlert,omitempty"`
	ArmedAt   string `json:"armedAt,omitempty"`
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

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "TELEGRAM_BOT_TOKEN env var required")
		os.Exit(1)
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	var chatID int64 = 584042304
	if chatIDStr != "" {
		fmt.Sscanf(chatIDStr, "%d", &chatID)
	}

	guard := &GuardState{
		Threshold: *sensitivity,
		Cooldown:  *cooldown,
		Token:     token,
		ChatID:    chatID,
	}

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

	// Start HTTP control API
	startHTTPServer(guard, *port)

	fmt.Fprintf(os.Stderr, "macguard: listening on :%d (disarmed)\n", *port)
	fmt.Fprintf(os.Stderr, "macguard: arm via POST /arm, disarm via POST /disarm\n")

	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "\nmacguard: shutting down")
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
			guard.MagEWMA = 0.05*mag + 0.95*guard.MagEWMA
		}

		if guard.Armed {
			if guard.MagEWMA >= guard.Threshold {
				if !guard.Moving {
					guard.Moving = true
				}
				// Send alert if cooldown has elapsed
				if time.Since(guard.LastAlert) >= guard.Cooldown {
					mag := guard.MagEWMA
					token := guard.Token
					chatID := guard.ChatID
					guard.LastAlert = time.Now()
					guard.mu.Unlock()
					go sendMovementAlert(token, chatID, mag)
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
		guard.mu.Lock()
		guard.Armed = true
		guard.Moving = false
		guard.MagEWMA = 0
		guard.ArmedAt = time.Now()
		guard.mu.Unlock()

		fmt.Fprintf(os.Stderr, "macguard: ARMED\n")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "armed"})
	})

	mux.HandleFunc("POST /disarm", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		guard.Armed = false
		guard.Moving = false
		guard.mu.Unlock()

		fmt.Fprintf(os.Stderr, "macguard: DISARMED\n")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "disarmed"})
	})

	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		guard.mu.Lock()
		resp := StatusResponse{
			Magnitude: guard.MagEWMA,
		}
		if guard.Armed {
			resp.Status = "armed"
			resp.Moving = guard.Moving
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
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"city":    geo.City,
			"region":  geo.Region,
			"country": geo.Country,
			"isp":     geo.ISP,
			"lat":     geo.Lat,
			"lon":     geo.Lon,
		})
	})

	go http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
