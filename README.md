# macguard

Theft-detection daemon for Apple Silicon Macs. Monitors the built-in accelerometer for physical movement and sends Telegram alerts with precise location.

## Features

- Accelerometer-based movement detection (EWMA threshold, hysteresis)
- Remote arm/disarm via HTTP API (designed for Telegram bot integration)
- Precise location via CoreLocation (Wi-Fi positioning, ~30-50m accuracy)
- IP geolocation fallback (city-level)
- Telegram alerts with location pin when movement detected
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
swiftc -o locate locate.swift -framework CoreLocation \
  -Xlinker -sectcreate -Xlinker __TEXT -Xlinker __info_plist -Xlinker Info.plist
```

### 2. CoreLocation permissions

Create the app bundle and sign it (required for location permissions):

```bash
mkdir -p Locate.app/Contents/MacOS
cp locate Locate.app/Contents/MacOS/locate
cp Info.plist Locate.app/Contents/Info.plist
codesign --force --sign - --entitlements entitlements.plist --deep Locate.app
open Locate.app  # triggers permission dialog - click Allow
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

## HTTP API

- `POST /arm` - arm the alarm
- `POST /disarm` - disarm
- `GET /status` - current state, magnitude, last alert
- `GET /location` - precise location (CoreLocation with IP fallback)

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

- `--port=8421` - HTTP control port
- `--sensitivity=0.045` - EWMA movement threshold (g)
- `--cooldown=60s` - minimum time between alerts
