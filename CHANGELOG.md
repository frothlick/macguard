# Changelog

## v0.5.0 — 2026-03-27

### Intruder Camera Capture
- Captures 5 photos at 1-second intervals on movement alert via CameraSnap.app (Swift AVFoundation helper)
- Simultaneous audio recording sent as Telegram voice message
- Photos and audio sent directly to Telegram

### Lid-Open Detection
- If lid is closed when alarm triggers, waits up to 5 minutes for intruder to open it
- 2-second delay after lid open to ensure face is in frame before capturing

### Remote Camera & Video via Telegram
- `/photo [N]` — capture N photos + audio on demand (default 3, max 10)
- `/video [N]` — record N-second video with audio (default 10s, max 60s)
- `sendTelegramPhoto`, `sendTelegramVideo`, `sendTelegramVoice` for multipart file uploads

### Intruder Warning Popup
- macOS dialog displayed on screen after alarm capture warning the intruder

### New Components
- `CameraSnap.app` — Swift helper for burst photo capture, audio recording, and video recording
- `camerasnap.swift` — source for CameraSnap.app with `--video` mode support

## v0.4.0 — 2026-03-24

### Security Hardening
- Dashboard bound to localhost only
- Credential masking in settings API
- Input validation on all endpoints
- Removed hardcoded Telegram token

### Dashboard Improvements
- AC power markers and battery tracking
- Responsive grid layout
- Custom alarm sound selection (Alert, Evacuation, Intruder, Klaxon, Siren)
- AC disconnect alarm option

## v0.3.0 — 2026-03-22

### Independent Guard Modes
- Local Guard and Geo Guard are now fully independent — arm one, both, or neither
- Each mode has its own card with description, status badge, and arm/disarm button
- No more confusing dropdown or shared countdown
- Telegram commands updated: `/arm`, `/arm_geo`, `/arm_both`, `/disarm`

### VPN Detection
- Detects VPN by comparing GPS coordinates vs IP geolocation (>100km = VPN)
- Shows actual location (reverse geocoded from GPS) as primary
- VPN exit location shown below with red VPN badge
- Uses Nominatim reverse geocoding for accurate location names from GPS

### Dashboard Improvements
- Avg line changed from green to solid gray; Peak line is opaque dashed orange
- Custom canvas legend icons: solid line with dot (Average), dashed line (Peak), hatched box (Lid closed)
- Lid-closed zones shown with diagonal gray hatching pattern instead of red bar
- Tooltips now show all sensor data: avg, peak, tilt, lid angle, lid state
- Zone legend at bottom updated to match chart hatching style
- About tab in Settings with author info and GitHub link
- Removed delay input from main UI (uses default from Settings)

### Zone Classification
- Tilt-based lap detection: laptop tilted >15° with peak activity classified as "lap"
- Fixes resting misclassification during active desk work (typing)

## v0.2.0 — 2026-03-22

### Dashboard
- Embedded single-page web dashboard at `http://localhost:8421`
- Chart.js movement visualization with 6 granularities: minute (60-min rolling), 24h, hour (4h), daypart (6x4h), day (7-day week), week (monthly)
- Zone-colored background segments: resting (gray), desk (green), lap (blue), motion (orange), impact (red)
- Lid-closed periods shown with diagonal hatch pattern overlay
- Sunrise/sunset markers with sun icons
- Custom legend with line-with-dot for Average, dashed line for Peak, and hatched box for Lid closed
- Detailed tooltips on hover: avg, peak, tilt, lid angle, lid state
- Dark/light theme toggle
- Leaflet.js map with location history markers
- Settings UI with tabs for General, Telegram, and Email configuration
- Real-time status polling (2s) and chart auto-refresh (60s)

### Lid Tracking
- Lid open/closed detection via `ioreg AppleClamshellState`
- Lid angle sensor (0-180) via IOKit HID (helpers/lidangle)
- Tilt angle tracking with baseline calibration
- Uses `*bool` for lid field to distinguish "unknown" (pre-tracking records) from "closed"
- Self-healing cleanup of legacy records that had false lid data baked in

### Zone Classification
- Improved classification using peak values: typing with low avg but peak >= 0.01g classified as "desk" instead of "resting"
- Tilt-based classification: tilted laptop (>15) with activity classified as "lap" (detects bed/couch use)
- Both Go backend and JS frontend use matching classification logic

### Alerting
- Email alerting via SMTP (port 465 implicit TLS, port 587 STARTTLS)
- CoreLocation integration for precise GPS location (via locate.swift helper)
- IP geolocation fallback via ip-api.com
- Geo-fence mode with haversine distance check (alerts if >50m from anchor)
- Location history with deduplication (~100m) and 500-entry cap

### Training Mode
- Per-second recording toggle for movement classification training
- Captures avg, peak, lid state, lid angle, tilt per second
- Separate session files: `training-YYYYMMDD-HHMMSS.json`
- Dashboard Rec button with elapsed time display

### Other
- Screen capture helper (capture.swift)
- Calibration endpoint for accelerometer baseline
- Persistent user settings (settings.json)

## v0.1.0 — 2026-03-21

- Initial release: theft-detection daemon for Apple Silicon Macs
- Accelerometer monitoring with EWMA smoothing and hysteresis
- Telegram bot integration for arm/disarm/status commands
- LaunchDaemon with RunAtLoad and KeepAlive
- Per-minute movement logging to `~/.macguard/YYYY-MM-DD.json`
