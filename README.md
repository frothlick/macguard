# macguard

Theft-detection daemon for Apple Silicon Macs. Monitors the built-in accelerometer for physical movement and sends Telegram alerts with precise location, intruder photos, audio, and video.

## Features

- Accelerometer-based movement detection (EWMA threshold, hysteresis)
- Independent Local Guard (accelerometer) and Geo Guard (geofence) modes
- Telegram bot with full remote control (arm, disarm, status, location, camera, video)
- Intruder camera capture: 5 photos + audio recording on movement alert
- Remote photo and video capture via Telegram (`/photo`, `/video`)
- Lid-open detection: waits for intruder to open lid, then captures with delay
- Intruder warning popup displayed on screen after capture
- Precise location via CoreLocation (Wi-Fi positioning, ~30-50m accuracy)
- IP geolocation fallback with VPN detection
- Web dashboard at `localhost:8421` with movement charts, location map, and settings
- Runs as a macOS launch daemon (starts on boot)

## Requirements

- Apple Silicon Mac (M1/M2/M3/M4)
- macOS 14+
- Telegram bot token (create one via [@BotFather](https://t.me/BotFather))
- Root privileges (accelerometer access requires IOKit HID)

## Setup

### 1. Build

```bash
go build -o macguard .

# Location helper
swiftc -o locate locate.swift -framework CoreLocation \
  -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist -Xlinker Info.plist

# Camera/audio/video helper
swiftc -o CameraSnap.app/Contents/MacOS/camerasnap camerasnap.swift \
  -framework AVFoundation -framework AppKit \
  -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist \
  -Xlinker CameraSnap.app/Contents/Info.plist
```

### 2. Permissions

```bash
# CoreLocation
mkdir -p Locate.app/Contents/MacOS
cp locate Locate.app/Contents/MacOS/locate
cp Info.plist Locate.app/Contents/Info.plist
codesign --force --sign - --entitlements entitlements.plist --deep Locate.app
open Locate.app  # triggers permission dialog - click Allow

# Camera & Microphone
./CameraSnap.app/Contents/MacOS/camerasnap /tmp/test 1 1000  # click Allow on both prompts
```

### 3. Configure Telegram

Edit `com.frothlick.macguard.plist` and replace:
- `YOUR_BOT_TOKEN_HERE` with your Telegram bot token
- `YOUR_CHAT_ID_HERE` with your Telegram chat ID

### 4. Install as launch daemon

```bash
sudo cp com.frothlick.macguard.plist /Library/LaunchDaemons/
sudo chown root:wheel /Library/LaunchDaemons/com.frothlick.macguard.plist
sudo launchctl load /Library/LaunchDaemons/com.frothlick.macguard.plist
```

## Telegram Commands

| Command | Description |
|---------|-------------|
| `/arm` | Arm local guard |
| `/arm_geo` | Arm geo guard |
| `/arm_both` | Arm both guards |
| `/disarm` | Disarm all |
| `/status` | Show guard status |
| `/location` | Send current location |
| `/photo [N]` | Capture N photos + audio (default 3, max 10) |
| `/video [N]` | Record N-second video (default 10s, max 60s) |
| `/msg [text]` | Display message on Mac screen |
| `/help` | Show available commands |

## HTTP API

- `POST /arm` — arm local guard
- `POST /arm_geo` — arm geo guard
- `POST /arm_both` — arm both guards
- `POST /disarm` — disarm
- `GET /status` — current state, magnitude, last alert
- `GET /location` — precise location (CoreLocation with IP fallback)

## Usage

```bash
# Manual start
export TELEGRAM_BOT_TOKEN="your-token"
sudo -E ./macguard

# Arm/disarm
curl -X POST localhost:8421/arm
curl -X POST localhost:8421/disarm
curl localhost:8421/status
curl localhost:8421/location
```

## Flags

- `--port=8421` — HTTP control port
- `--sensitivity=0.045` — EWMA movement threshold (g)
- `--cooldown=60s` — minimum time between alerts
