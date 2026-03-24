package main

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MacGuard</title>
<link rel="icon" href="/favicon.ico" type="image/x-icon">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9/dist/leaflet.css"/>
<script src="https://unpkg.com/leaflet@1.9/dist/leaflet.js"></script>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  :root { --bg: #0a0a0a; --card: #141414; --border: #222; --text: #e0e0e0; --muted: #888; --dim: #555; --gridline: #2a2a2a; --accent: #00ffaa; --picker-bg: #222; --picker-border: #333; }
  body.light { --bg: #f5f5f5; --card: #fff; --border: #ddd; --text: #222; --muted: #666; --dim: #999; --gridline: #eee; --accent: #00aa77; --picker-bg: #eee; --picker-border: #ccc; }
  body { background: var(--bg); color: var(--text); font-family: -apple-system, system-ui, sans-serif; padding: 20px; transition: background 0.3s, color 0.3s; }
  .header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 20px; }
  .header-left { display: flex; align-items: center; gap: 12px; }
  .header-left img { height: 48px; width: 48px; object-fit: contain; }
  .header-left h1 { color: var(--accent); font-size: 1.6em; margin-bottom: 0; }
  .header-left .subtitle { color: var(--muted); font-size: 0.85em; }
  .theme-toggle { background: var(--picker-bg); border: 1px solid var(--picker-border); color: var(--muted); border-radius: 8px; padding: 6px 14px; cursor: pointer; font-size: 0.85em; }
  .theme-toggle:hover { filter: brightness(1.2); }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
  .grid3 { display: grid; grid-template-columns: minmax(0, 200px) 1fr minmax(0, 200px); gap: 16px; margin-bottom: 16px; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 16px; }
  .card.full { grid-column: 1 / -1; }
  .card h2 { color: var(--muted); font-size: 0.8em; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 10px; }
  .stat { font-size: 2em; font-weight: 700; color: var(--accent); }
  .status-badge { display: inline-block; padding: 4px 12px; border-radius: 12px; font-weight: 600; font-size: 0.85em; }
  .status-armed { background: #ff330033; color: #ff5544; border: 1px solid #ff3300; }
  .status-disarmed { background: #00ff8833; color: var(--accent); border: 1px solid #00ff88; }
  .btn { padding: 8px 20px; border-radius: 8px; border: 1px solid var(--picker-border); font-weight: 600; cursor: pointer; font-size: 0.9em; transition: all 0.2s; }
  .btn:hover { filter: brightness(1.2); }
  .btn-arm { background: #ff330033; color: #ff5544; border-color: #ff3300; }
  .btn-arm:hover { background: #ff330066; }
  .btn-disarm { background: #00ff8833; color: var(--accent); border-color: #00ff88; }
  .btn-disarm:hover { background: #00ff8866; }
  .btn-loc { background: #3388ff33; color: #5599ff; border-color: #3388ff; }
  .btn-loc:hover { background: #3388ff66; }
  .controls { display: flex; gap: 10px; margin-top: 10px; }
  .gran-picker { display: flex; gap: 4px; }
  .gran-picker button { background: var(--picker-bg); color: var(--muted); border: 1px solid var(--picker-border); border-radius: 6px; padding: 5px 12px; cursor: pointer; font-size: 0.78em; }
  .gran-picker button:hover { filter: brightness(1.15); }
  .gran-picker button.active { background: #00ffaa22; color: var(--accent); border-color: var(--accent); }
  .btn-nav { background: var(--picker-bg); color: var(--muted); border: 1px solid var(--picker-border); border-radius: 6px; padding: 5px 10px; cursor: pointer; font-size: 0.9em; }
  .btn-nav:hover { filter: brightness(1.15); }
  canvas { max-height: 300px; }
  .zone-legend { display: flex; gap: 16px; margin-top: 8px; flex-wrap: wrap; }
  .zone-legend span { font-size: 0.75em; color: var(--muted); }
  .zone-legend span::before { content: ''; display: inline-block; width: 10px; height: 10px; border-radius: 2px; margin-right: 4px; vertical-align: middle; }
  .z-resting::before { background: #999; }
  .z-desk::before { background: #44bb66; }
  .z-lap::before { background: #3399ff; }
  .z-motion::before { background: #ffaa00; }
  .z-impact::before { background: #ff5500; }
  #map { height: 100%; min-height: 200px; border-radius: 8px; }
  .loc-info { color: var(--muted); font-size: 0.85em; margin-top: 8px; }
  @media (max-width: 700px) { .grid { grid-template-columns: 1fr; } .grid3 { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div class="header">
  <div class="header-left">
    <div>
      <h1>MacGuard</h1>
      <span class="subtitle">by alexander@wipf.com</span>
    </div>
  </div>
  <div style="display:flex; gap:8px; align-items:center">
    <button class="theme-toggle" onclick="toggleTheme()" id="theme-btn">Light</button>
    <button class="theme-toggle" onclick="openSettings()" title="Settings" style="font-size:1.1em; padding:4px 10px">&#9881;</button>
  </div>
</div>

<div id="settings-overlay" style="display:none; position:fixed; top:0; left:0; right:0; bottom:0; background:rgba(0,0,0,0.6); z-index:1000; justify-content:center; align-items:center">
  <div class="card" style="width:520px; max-width:92vw; height:480px; position:relative; padding:24px; display:flex; flex-direction:column">
    <button onclick="closeSettings()" style="position:absolute; top:12px; right:16px; background:none; border:none; color:var(--muted); font-size:1.4em; cursor:pointer">&times;</button>
    <h2 style="margin-bottom:16px; font-size:0.95em">Settings</h2>

    <div class="settings-tabs" style="display:flex; gap:0; margin-bottom:20px; border-bottom:1px solid var(--border)">
      <button onclick="setSettingsTab('general')" id="tab-general" style="background:none; border:none; border-bottom:2px solid var(--accent); color:var(--text); padding:8px 20px; cursor:pointer; font-size:0.85em; font-weight:600">General</button>
      <button onclick="setSettingsTab('telegram')" id="tab-telegram" style="background:none; border:none; border-bottom:2px solid transparent; color:var(--muted); padding:8px 20px; cursor:pointer; font-size:0.85em; font-weight:600">Telegram</button>
      <button onclick="setSettingsTab('email')" id="tab-email" style="background:none; border:none; border-bottom:2px solid transparent; color:var(--muted); padding:8px 20px; cursor:pointer; font-size:0.85em; font-weight:600">Email</button>
      <button onclick="setSettingsTab('alarm')" id="tab-alarm" style="background:none; border:none; border-bottom:2px solid transparent; color:var(--muted); padding:8px 20px; cursor:pointer; font-size:0.85em; font-weight:600">Alarm</button>
      <button onclick="setSettingsTab('about')" id="tab-about" style="background:none; border:none; border-bottom:2px solid transparent; color:var(--muted); padding:8px 20px; cursor:pointer; font-size:0.85em; font-weight:600">About</button>
    </div>

    <div style="flex:1; overflow-y:auto; min-height:0">
    <div id="tab-content-general">
      <div style="margin-bottom:20px">
        <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Default Arm Delay</label>
        <div style="display:flex; align-items:center; gap:6px; margin-top:8px">
          <input id="set-delay" type="number" min="0" max="300" value="0" style="background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:90px; font-size:0.9em">
          <span style="font-size:0.82em; color:var(--dim)">seconds</span>
        </div>
      </div>

      <div style="margin-bottom:20px">
        <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Calibration</label>
        <div style="font-size:0.8em; color:var(--dim); margin-top:6px; line-height:1.5">
          1. Place your Mac on a standard-height desk (72&ndash;76 cm, DIN norm) and sit in your normal position.<br>
          2. Open the lid to your comfortable viewing angle.<br>
          3. Do not touch or move the laptop during calibration.<br>
          The process takes ~3 seconds and records accelerometer baseline, tilt offset, and your desk lid angle for accurate zone classification.
        </div>
        <div style="display:flex; align-items:center; gap:12px; margin-top:8px">
          <button class="btn btn-loc" onclick="doCalibrate()" id="calib-btn" style="font-size:0.82em; padding:8px 16px">Calibrate</button>
          <span id="baseline-info" style="font-size:0.82em; color:var(--dim)"></span>
        </div>
        <div id="calib-details" style="font-size:0.78em; color:var(--dim); margin-top:8px; display:none"></div>
      </div>
    </div>

    <div id="tab-content-telegram" style="display:none">
      <div style="margin-bottom:20px">
        <label style="display:flex; align-items:center; gap:10px; cursor:pointer">
          <input type="checkbox" id="set-telegram" onchange="toggleTelegramFields()" style="width:18px; height:18px">
          <span style="font-size:0.92em; font-weight:500">Enable Telegram notifications</span>
        </label>
      </div>

      <div id="telegram-fields" style="display:none; border:1px dashed var(--border); border-radius:8px; padding:16px; margin-bottom:20px">
        <div style="margin-bottom:14px">
          <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Chat ID</label>
          <input id="set-chat-id" type="text" placeholder="Send /start to @MacGuard_bot" style="display:block; margin-top:6px; background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:100%; font-size:0.9em">
          <span style="font-size:0.72em; color:var(--dim)">Message /start to @MacGuard_bot to get your ID</span>
        </div>
      </div>
    </div>

    <div id="tab-content-email" style="display:none">
      <div style="margin-bottom:20px">
        <label style="display:flex; align-items:center; gap:10px; cursor:pointer">
          <input type="checkbox" id="set-email" onchange="toggleEmailFields()" style="width:18px; height:18px">
          <span style="font-size:0.92em; font-weight:500">Enable Email notifications</span>
        </label>
      </div>

      <div id="email-fields" style="display:none; border:1px dashed var(--border); border-radius:8px; padding:16px; margin-bottom:20px">
        <div style="margin-bottom:14px">
          <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Alert Recipient</label>
          <input id="set-email-addr" type="email" placeholder="you@example.com" style="display:block; margin-top:6px; background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:100%; font-size:0.9em">
        </div>

        <div style="margin-bottom:14px">
          <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">SMTP Server</label>
          <input id="set-smtp-host" type="text" placeholder="smtp.gmail.com:465" style="display:block; margin-top:6px; background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:100%; font-size:0.9em">
        </div>

        <div style="display:flex; gap:8px">
          <div style="flex:1">
            <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">SMTP User</label>
            <input id="set-smtp-user" type="text" placeholder="user@example.com" style="display:block; margin-top:6px; background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:100%; font-size:0.9em">
          </div>
          <div style="flex:1">
            <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">SMTP Password</label>
            <input id="set-smtp-pass" type="password" placeholder="password" style="display:block; margin-top:6px; background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; width:100%; font-size:0.9em">
          </div>
        </div>
        <span style="font-size:0.72em; color:var(--dim); display:block; margin-top:8px">Use port 465 (SSL) or 587 (TLS)</span>
      </div>
    </div>

    <div id="tab-content-alarm" style="display:none">
      <div style="margin-bottom:20px">
        <label style="display:flex; align-items:center; gap:10px; cursor:pointer">
          <input type="checkbox" id="set-alarm-enabled" style="width:18px; height:18px">
          <span style="font-size:0.92em; font-weight:500">Enable alarm sound</span>
        </label>
        <span style="font-size:0.72em; color:var(--dim); margin-left:28px">Plays an audible alarm through the laptop speakers when triggered</span>
      </div>

      <div style="margin-bottom:20px">
        <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Movement Alarm Sound</label>
        <div style="display:flex; align-items:center; gap:8px; margin-top:8px">
          <select id="set-alarm-sound" style="background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; font-size:0.9em; flex:1">
            <option value="Siren" selected>Siren</option><option value="Klaxon">Klaxon</option><option value="Alert">Alert</option><option value="Intruder">Intruder</option><option value="Evacuation">Evacuation</option>
          </select>
          <button class="btn btn-loc" onclick="testAlarmSound('set-alarm-sound')" style="font-size:0.82em; padding:8px 12px; white-space:nowrap">Preview</button>
        </div>
      </div>

      <div style="margin-bottom:20px">
        <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">Geo-Fence Alarm Sound</label>
        <div style="display:flex; align-items:center; gap:8px; margin-top:8px">
          <select id="set-geo-alarm-sound" style="background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; font-size:0.9em; flex:1">
            <option value="Siren">Siren</option><option value="Klaxon">Klaxon</option><option value="Alert">Alert</option><option value="Intruder" selected>Intruder</option><option value="Evacuation">Evacuation</option>
          </select>
          <button class="btn btn-loc" onclick="testAlarmSound('set-geo-alarm-sound')" style="font-size:0.82em; padding:8px 12px; white-space:nowrap">Preview</button>
        </div>
      </div>

      <div style="margin-bottom:20px">
        <label style="display:flex; align-items:center; gap:10px; cursor:pointer">
          <input type="checkbox" id="set-ac-alarm" style="width:18px; height:18px">
          <span style="font-size:0.92em; font-weight:500">Alarm on AC disconnect</span>
        </label>
        <span style="font-size:0.72em; color:var(--dim); margin-left:28px">Play sound when the charger is unplugged while armed</span>
      </div>

      <div style="margin-bottom:20px">
        <label style="font-size:0.82em; color:var(--muted); text-transform:uppercase; letter-spacing:0.5px">AC Disconnect Sound</label>
        <div style="display:flex; align-items:center; gap:8px; margin-top:8px">
          <select id="set-ac-alarm-sound" style="background:var(--picker-bg); color:var(--text); border:1px solid var(--picker-border); border-radius:6px; padding:8px 12px; font-size:0.9em; flex:1">
            <option value="Funk" selected>Funk</option><option value="Basso">Basso</option><option value="Blow">Blow</option><option value="Bottle">Bottle</option><option value="Frog">Frog</option><option value="Glass">Glass</option><option value="Hero">Hero</option><option value="Morse">Morse</option><option value="Ping">Ping</option><option value="Pop">Pop</option><option value="Purr">Purr</option><option value="Sosumi">Sosumi</option><option value="Submarine">Submarine</option><option value="Tink">Tink</option>
            <option value="Siren">Siren</option><option value="Klaxon">Klaxon</option><option value="Alert">Alert</option><option value="Intruder">Intruder</option><option value="Evacuation">Evacuation</option>
          </select>
          <button class="btn btn-loc" onclick="testAlarmSound('set-ac-alarm-sound')" style="font-size:0.82em; padding:8px 12px; white-space:nowrap">Preview</button>
        </div>
      </div>
    </div>

    <div id="tab-content-about" style="display:none; text-align:center; padding:20px 0">
      <div style="font-size:1.4em; font-weight:700; margin-bottom:4px">MacGuard</div>
      <div style="font-size:0.82em; color:var(--muted); margin-bottom:20px">Theft-detection daemon for Apple Silicon Macs</div>
      <div style="margin-bottom:16px; display:flex; align-items:center; justify-content:center; gap:12px">
        <img src="/favicon.ico" alt="wipf.com" style="width:40px; height:40px; border-radius:6px">
        <div>
          <div style="font-size:0.9em; font-weight:500">Alexander Wipf</div>
          <a href="mailto:alexander@wipf.com" style="font-size:0.82em; color:var(--accent); text-decoration:none">alexander@wipf.com</a>
        </div>
      </div>
      <div style="margin-bottom:20px">
        <a href="https://github.com/frothlick/macguard" target="_blank" style="font-size:0.85em; color:var(--accent); text-decoration:none">github.com/frothlick/macguard</a>
      </div>
      <div style="font-size:0.72em; color:var(--dim)">v0.4.0</div>
    </div>

    </div>
    <button class="btn btn-disarm" onclick="saveSettingsUI()" style="width:100%; padding:10px; flex-shrink:0; margin-top:12px">Save</button>
    <div id="settings-msg" style="text-align:center; font-size:0.82em; margin-top:10px; color:var(--accent)"></div>
  </div>
</div>

<div class="grid3">
  <div class="card" id="card-guard" style="transition: opacity 0.3s">
    <h2>Arm Status</h2>
    <div style="font-size:0.82em; color:var(--muted); margin-bottom:10px">Arm to send alerts when your Mac is physically moved</div>
    <div id="local-status"></div>
    <div id="local-controls" class="controls" style="margin-top:10px"></div>
    <div style="border-top:1px solid #333; margin:14px 0"></div>
    <div style="font-size:0.82em; color:var(--muted); margin-bottom:10px">Arm to send alerts when your Mac leaves this location</div>
    <div id="geo-status"></div>
    <div id="geo-controls" class="controls" style="margin-top:10px"></div>
  </div>
  <div class="card">
    <h2>Map</h2>
    <div id="map"></div>
  </div>
  <div class="card">
    <h2>Live Data</h2>
    <div class="stat" id="magnitude">...</div>
    <div id="zone-label" style="color:var(--muted); font-size:0.85em; margin-top:4px"></div>
    <div id="mag-baseline-info" style="font-size:0.72em; color:var(--dim); margin-top:4px"></div>
    <div id="ac-status" style="font-size:0.78em; margin-top:6px"></div>
    <div style="border-top:1px solid #333; margin:14px 0"></div>
    <h2>Location</h2>
    <div id="loc-current" class="loc-info">...</div>
    <div class="controls">
      <button class="btn btn-loc" onclick="refreshLocation()">Refresh</button>
    </div>
    <div id="loc-recent" style="margin-top:8px; border-top:1px solid #222; padding-top:6px; font-size:0.78em; color:#777; max-height:90px; overflow-y:auto"></div>
  </div>
</div>

<div class="grid">
  <div class="card full">
    <div style="display:flex; align-items:center; justify-content:space-between; margin-bottom:12px; flex-wrap:wrap; gap:8px">
      <h2 style="margin:0">Movement</h2>
      <button id="rec-btn" onclick="toggleTraining()" style="background:var(--picker-bg); color:var(--muted); border:1px solid var(--picker-border); border-radius:6px; padding:5px 12px; cursor:pointer; font-size:0.78em; display:flex; align-items:center; gap:5px">
        <span id="rec-dot" style="width:8px; height:8px; border-radius:50%; background:#ff3300; display:none"></span>
        <span id="rec-label">Rec</span>
      </button>
      <div class="gran-nav">
        <button class="btn-nav" onclick="navPrev()">◄</button>
        <span id="nav-label" style="color:#888; min-width:140px; text-align:center; display:inline-block"></span>
        <button class="btn-nav" onclick="navNext()">►</button>
      </div>
      <div class="gran-picker" id="gran-picker">
        <button onclick="setGran('minute')" class="active">Minute</button>
        <button onclick="setGran('24h')">24h</button>
        <button onclick="setGran('hour')">Hour</button>
        <button onclick="setGran('daypart')">Daypart</button>
        <button onclick="setGran('day')">Day</button>
        <button onclick="setGran('week')">Week</button>
      </div>
    </div>
    <canvas id="chart"></canvas>
    <div class="zone-legend">
      <span class="z-resting">resting</span>
      <span class="z-desk">desk use</span>
      <span class="z-lap">lap use</span>
      <span class="z-motion">in motion</span>
      <span class="z-impact">impact</span>
      <span style="font-size:0.75em; color:var(--muted)"><canvas id="zone-lid-icon" width="12" height="12" style="vertical-align:middle; margin-right:4px"></canvas>lid closed</span>
    </div>
  </div>
</div>

<script>
const zoneColors = { resting: '#999999', desk: '#44bb66', lap: '#3399ff', motion: '#ffaa00', impact: '#ff5500' };
let map, markers = [], tileLayer;
const tiles = {
  dark: { url: 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png', attr: '&copy; OpenStreetMap &copy; CARTO' },
  light: { url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', attr: '&copy; <a href="https://openstreetmap.org">OpenStreetMap</a> contributors' }
};
let isDark = localStorage.getItem('macguard-theme') !== 'light';

function applyTheme() {
  document.body.classList.toggle('light', !isDark);
  document.getElementById('theme-btn').textContent = isDark ? 'Light' : 'Dark';
  localStorage.setItem('macguard-theme', isDark ? 'dark' : 'light');
  if (map && tileLayer) {
    map.removeLayer(tileLayer);
    const t = isDark ? tiles.dark : tiles.light;
    tileLayer = L.tileLayer(t.url, { attribution: t.attr, maxZoom: 19 }).addTo(map);
  }
  if (chart) loadChart();
}

function toggleTheme() {
  isDark = !isDark;
  applyTheme();
}

function getCSS(v) { return getComputedStyle(document.body).getPropertyValue(v).trim(); }

var gLidAngleBaseline = 0;
function classifyZone(mag, peak, tilt, lidAngle, lidOpen) {
  var tilted = (tilt || 0) > 2;
  if (mag >= 0.040) return 'impact';
  if (mag >= 0.015) return 'motion';
  if (lidOpen) {
    if (mag >= 0.004 && tilted) return 'lap';
    if (mag >= 0.004) return 'desk';
    if (tilted && (peak || 0) >= 0.01) return 'lap';
    if (mag >= 0.001) return 'desk';
    if ((peak || 0) >= 0.01) return 'desk';
  }
  if (!lidOpen) {
    if (mag >= 0.004) return 'motion';
    if ((peak || 0) >= 0.05) return 'motion';
  }
  return 'resting';
}

// --- Controls ---
function estimateHeight(lidAngle) {
  // Higher desk lid angle = taller person (needs to tilt screen back more)
  // ~100° ≈ 160cm, ~111° ≈ 180cm, ~120° ≈ 200cm
  if (lidAngle <= 0) return '';
  var cm = Math.round(160 + (lidAngle - 100) * 2);
  if (cm < 150) cm = 150;
  if (cm > 210) cm = 210;
  return '~' + cm + 'cm';
}

function showCalibDetails() {
  var el = document.getElementById('calib-details');
  if (!el) return;
  fetch('/status').then(function(r) { return r.json(); }).then(function(d) {
    if (d.baseline > 0) {
      var lines = [];
      lines.push('Noise floor: ' + d.baseline.toFixed(6) + 'g');
      if (d.lidAngleBaseline > 0) {
        lines.push('Desk lid angle: ' + d.lidAngleBaseline.toFixed(0) + '\u00b0');
        var h = estimateHeight(d.lidAngleBaseline);
        if (h) lines.push('Estimated height: ' + h);
      }
      el.innerHTML = lines.join(' &middot; ');
      el.style.display = 'block';
    }
  });
}

async function doCalibrate() {
  document.getElementById('calib-btn').textContent = 'Calibrating...';
  var bi = document.getElementById('baseline-info');
  if (bi) { bi.textContent = '---'; bi.style.color = 'var(--dim)'; }
  await fetch('/calibrate', { method: 'POST' });
  setTimeout(function() { document.getElementById('calib-btn').textContent = 'Calibrate'; loadStatus(); showCalibDetails(); }, 3500);
}

async function doArm(mode) {
  const s = await fetch('/settings').then(r => r.json()).catch(function() { return {}; });
  const delay = s.defaultDelay || 0;
  await fetch('/arm', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mode: mode, delay: delay })
  });
  loadStatus();
}
async function doDisarm(mode) {
  await fetch('/disarm', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mode: mode })
  });
  loadStatus();
}

// --- Status ---
async function loadStatus() {
  try {
    const r = await fetch('/status');
    const d = await r.json();
    // Local Guard
    var localStatus = document.getElementById('local-status');
    var localCtrl = document.getElementById('local-controls');
    if (d.local.status === 'armed') {
      localStatus.innerHTML = '<span class="status-badge status-armed">ARMED</span>' + (d.moving ? ' <span style="color:#ff5500; font-weight:700">MOVING</span>' : '');
      localCtrl.innerHTML = '<button class="btn btn-disarm" onclick="doDisarm(\'move\')" style="flex:1">Local Disarm</button>';
    } else if (d.local.status === 'arming') {
      localStatus.innerHTML = '<span class="status-badge status-armed">ARMING ' + d.local.delay + 's</span>';
      localCtrl.innerHTML = '<button class="btn btn-disarm" onclick="doDisarm(\'move\')" style="flex:1">Cancel</button>';
    } else {
      localStatus.innerHTML = '';
      localCtrl.innerHTML = '<button class="btn btn-arm" onclick="doArm(\'move\')" style="flex:1">Local Arm</button>';
    }

    // Geo Guard
    var geoStatus = document.getElementById('geo-status');
    var geoCtrl = document.getElementById('geo-controls');
    if (d.geo.status === 'armed') {
      geoStatus.innerHTML = '<span class="status-badge status-armed">ARMED</span>' + (d.moving ? ' <span style="color:#ff5500; font-weight:700">MOVING</span>' : '');
      geoCtrl.innerHTML = '<button class="btn btn-disarm" onclick="doDisarm(\'geo\')" style="flex:1">Geo Disarm</button>';
    } else if (d.geo.status === 'arming') {
      geoStatus.innerHTML = '<span class="status-badge status-armed">ARMING ' + d.geo.delay + 's</span>';
      geoCtrl.innerHTML = '<button class="btn btn-disarm" onclick="doDisarm(\'geo\')" style="flex:1">Cancel</button>';
    } else {
      geoStatus.innerHTML = '';
      geoCtrl.innerHTML = '<button class="btn btn-arm" onclick="doArm(\'geo\')" style="flex:1">Geo Arm</button>';
    }

    const mag = d.magnitude || 0;
    document.getElementById('magnitude').textContent = d.calibrating ? 'calibrating...' : mag.toFixed(3) + 'g';
    const zone = classifyZone(mag);
    document.getElementById('zone-label').innerHTML = d.calibrating ? '' : '<span style="color:' + zoneColors[zone] + '">' + zone + '</span>';
    document.getElementById('mag-baseline-info').textContent = (d.baseline > 0) ? 'Calibrated' : 'Not calibrated';
    document.getElementById('mag-baseline-info').style.color = (d.baseline > 0) ? 'var(--accent)' : 'var(--dim)';
    if (d.lidAngleBaseline > 0) gLidAngleBaseline = d.lidAngleBaseline;
    document.getElementById('ac-status').innerHTML = d.acPower ? '<span style="color:#00ffaa">AC Power</span>' : '<span style="color:var(--dim)">Battery</span>';
  } catch(e) {}
}

// --- Location ---
async function refreshLocation() {
  document.getElementById('loc-current').textContent = 'Fetching...';
  try {
    const r = await fetch('/location');
    const d = await r.json();
    if (d.status === 'ok') {
      let txt = d.city || '';
      if (d.region) txt += ', ' + d.region;
      if (d.country) txt += ', ' + d.country;
      txt += ' (' + d.lat.toFixed(4) + ', ' + d.lon.toFixed(4) + ')';
      const precLabel = d.precise ? ' <span style="color:#00ffaa">[precise]</span>' : ' <span style="color:#ff9900">[approx]</span>';
      var html = txt + precLabel;
      if (d.vpn) {
        var vpnLoc = d.vpnCity || '';
        if (d.vpnRegion) vpnLoc += ', ' + d.vpnRegion;
        if (d.vpnCountry) vpnLoc += ', ' + d.vpnCountry;
        html += '<div style="margin-top:4px; font-size:0.8em"><span style="color:#ff5500; font-weight:600">VPN</span> <span style="color:var(--dim)">' + vpnLoc + '</span></div>';
      }
      document.getElementById('loc-current').innerHTML = html;
      userLat = d.lat; userLon = d.lon;
      if (map) map.setView([d.lat, d.lon], 14);
    } else {
      document.getElementById('loc-current').textContent = 'Unavailable';
    }
    loadLocations();
  } catch(e) {
    document.getElementById('loc-current').textContent = 'Error';
  }
}

async function loadLocations() {
  try {
    const r = await fetch('/locations');
    const locs = await r.json();
    if (!locs || !locs.length) return;

    // Clear old markers
    markers.forEach(m => map.removeLayer(m));
    markers = [];

    // Add markers + recent list
    const recent = locs.slice(-20).reverse();
    const recentEl = document.getElementById('loc-recent');
    const rows = [];
    recent.forEach((loc, i) => {
      const isPrecise = loc.precise;
      const markerColor = isPrecise ? (i === 0 ? '#00ffaa' : '#3388ff') : '#ff9900';
      const m = L.circleMarker([loc.lat, loc.lon], {
        radius: i === 0 ? 8 : (isPrecise ? 5 : 7),
        color: markerColor,
        fillColor: markerColor,
        fillOpacity: i === 0 ? 0.9 : (isPrecise ? 0.5 : 0.3),
        weight: isPrecise ? 2 : 1,
        dashArray: isPrecise ? null : '4',
      }).addTo(map);
      const label = loc.city || (loc.lat.toFixed(3) + ', ' + loc.lon.toFixed(3));
      const precTag = isPrecise ? 'precise' : 'approx';
      m.bindPopup(loc.time + '<br>' + label + '<br><em>' + precTag + '</em>');
      markers.push(m);

      if (i > 0 && i < 6) {
        const dot = isPrecise ? '<span style="color:#3388ff">●</span>' : '<span style="color:#ff9900">○</span>';
        rows.push(dot + ' ' + loc.time.slice(5) + ' — ' + label);
      }
    });
    recentEl.innerHTML = rows.length ? '<b style="color:#999">Recent</b><br>' + rows.join('<br>') : '';

    // Fit map to markers
    if (markers.length > 1) {
      const group = L.featureGroup(markers);
      map.fitBounds(group.getBounds().pad(0.2));
    } else if (markers.length === 1) {
      map.setView([recent[0].lat, recent[0].lon], 14);
    }
  } catch(e) {}
}

// --- Movement chart ---
let chart = null;
let gran = 'minute';
let cursor = new Date();
let userLat = 48.22, userLon = 9.01;
const dayparts = ['00-04','04-08','08-12','12-16','16-20','20-24'];

function dateStr(d) { return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0'); }
function addDays(d, n) { const r = new Date(d); r.setDate(r.getDate()+n); return r; }
function pad2(n) { return String(((n%60)+60)%60).padStart(2,'0'); }
function padH(n) { return String(((n%24)+24)%24).padStart(2,'0'); }
function timeToMin(t) { return parseInt(t.slice(0,2)) * 60 + parseInt(t.slice(3,5)); }

function getISOWeek(d) {
  const t = new Date(d.getTime());
  t.setHours(0,0,0,0);
  t.setDate(t.getDate() + 3 - (t.getDay() + 6) % 7);
  const w1 = new Date(t.getFullYear(), 0, 4);
  return 1 + Math.round(((t - w1) / 86400000 - 3 + (w1.getDay() + 6) % 7) / 7);
}

function getSunTimes(lat, lon, date) {
  const rad = Math.PI / 180;
  const doy = Math.floor((date - new Date(date.getFullYear(), 0, 0)) / 86400000);
  const B = 2 * Math.PI * (doy - 1) / 365;
  const EoT = 229.18 * (0.000075 + 0.001868*Math.cos(B) - 0.032077*Math.sin(B) - 0.014615*Math.cos(2*B) - 0.04089*Math.sin(2*B));
  const dec = 23.45 * Math.sin(2 * Math.PI * (284 + doy) / 365) * rad;
  const tz = -date.getTimezoneOffset() / 60;
  var noon = 12 - EoT/60 - lon/15 + tz;
  var cosHA = (Math.sin(-0.833 * rad) - Math.sin(lat * rad) * Math.sin(dec)) / (Math.cos(lat * rad) * Math.cos(dec));
  if (Math.abs(cosHA) > 1) return null;
  var HA = Math.acos(cosHA) / rad / 15;
  var rise = noon - HA, set = noon + HA;
  function fmt(h) { var hr = Math.floor(h); var mn = Math.round((h - hr) * 60); if (mn === 60) { hr++; mn = 0; } return padH(hr) + ':' + pad2(mn); }
  return { rise: fmt(rise), riseMin: Math.round(rise*60), noon: fmt(noon), noonMin: Math.round(noon*60), set: fmt(set), setMin: Math.round(set*60) };
}

function setGran(g) {
  gran = g;
  cursor = new Date();
  document.querySelectorAll('.gran-picker button').forEach(b => b.classList.toggle('active', b.textContent.toLowerCase() === g || (g === '24h' && b.textContent === '24h')));
  loadChart();
}

function navPrev() {
  if (gran === 'minute') cursor.setHours(cursor.getHours()-1);
  else if (gran === '24h') cursor = addDays(cursor, -1);
  else if (gran === 'hour') cursor.setHours(cursor.getHours()-4);
  else if (gran === 'daypart') cursor = addDays(cursor, -1);
  else if (gran === 'day') cursor = addDays(cursor, -7);
  else if (gran === 'week') cursor = addDays(cursor, -28);
  loadChart();
}

function navNext() {
  if (gran === 'minute') cursor.setHours(cursor.getHours()+1);
  else if (gran === '24h') cursor = addDays(cursor, 1);
  else if (gran === 'hour') cursor.setHours(cursor.getHours()+4);
  else if (gran === 'daypart') cursor = addDays(cursor, 1);
  else if (gran === 'day') cursor = addDays(cursor, 7);
  else if (gran === 'week') cursor = addDays(cursor, 28);
  loadChart();
}

async function fetchDay(date) {
  try {
    const r = await fetch('/activity?date=' + date);
    const d = await r.json();
    return d.records || [];
  } catch(e) { return []; }
}

async function fetchRange(from, to) {
  try {
    const r = await fetch('/activity/range?from=' + from + '&to=' + to);
    const days = await r.json();
    return days || [];
  } catch(e) { return []; }
}

function sunMarkersForRange(startMin, endMin, date) {
  const sun = getSunTimes(userLat, userLon, date);
  if (!sun) return [];
  const marks = [];
  [{min: sun.riseMin, label: sun.rise, icon: '\u2600\u2191', color: '#ffaa00'},
   {min: sun.noonMin, label: sun.noon, icon: '\u2600', color: '#ffdd00'},
   {min: sun.setMin, label: sun.set, icon: '\u2600\u2193', color: '#ff6600'}
  ].forEach(m => {
    if (m.min >= startMin && m.min <= endMin) marks.push(m);
  });
  return marks;
}

async function loadChart() {
  const navLabel = document.getElementById('nav-label');
  let labels = [], data = [], peakData = [], lidData = [], tiltData = [], lidAngleData = [], acData = [], batData = [], segments = [], sunMarks = [];

  if (gran === 'minute') {
    // Rolling 60-minute window ending at cursor
    const end = new Date(cursor);
    const start = new Date(end.getTime() - 3600000);
    const sd = dateStr(start), ed = dateStr(end);
    navLabel.textContent = sd.slice(5) + ' ' + padH(start.getHours()) + ':' + pad2(start.getMinutes()) + ' \u2014 ' + padH(end.getHours()) + ':' + pad2(end.getMinutes());

    const sMin = start.getHours() * 60 + start.getMinutes();
    const eMin = end.getHours() * 60 + end.getMinutes();

    let filtered = [];
    if (sd === ed) {
      const recs = await fetchDay(ed);
      filtered = recs.filter(function(r) { var m = timeToMin(r.t); return m >= sMin && m <= eMin; });
    } else {
      var r1 = await fetchDay(sd); var r2 = await fetchDay(ed);
      filtered = r1.filter(function(r) { return timeToMin(r.t) >= sMin; }).concat(r2.filter(function(r) { return timeToMin(r.t) <= eMin; }));
    }
    labels = filtered.map(function(r) { return r.t; });
    data = filtered.map(function(r) { return r.avg; });
    peakData = filtered.map(function(r) { return r.peak || 0; });
    lidData = filtered.map(function(r) { return r.lid; });
    tiltData = filtered.map(function(r) { return r.tilt || 0; });
    lidAngleData = filtered.map(function(r) { return r.lidAngle || 0; });
    acData = filtered.map(function(r) { return r.ac; });
    batData = filtered.map(function(r) { return r.bat != null ? r.bat : null; });
    segments = filtered.map(function(r) { return zoneColors[r.zone || classifyZone(r.avg, r.peak, r.tilt, r.lidAngle, r.lidAngle > 0)] || '#999'; });
    sunMarks = sunMarkersForRange(sMin < eMin ? sMin : sMin, sMin < eMin ? eMin : eMin + 1440, end);

  } else if (gran === '24h') {
    // Full day view - all minute records for one day
    var date = dateStr(cursor);
    navLabel.textContent = date;
    var records = await fetchDay(date);
    labels = records.map(function(r) { return r.t; });
    data = records.map(function(r) { return r.avg; });
    peakData = records.map(function(r) { return r.peak || 0; });
    lidData = records.map(function(r) { return r.lid; });
    tiltData = records.map(function(r) { return r.tilt || 0; });
    lidAngleData = records.map(function(r) { return r.lidAngle || 0; });
    acData = records.map(function(r) { return r.ac; });
    batData = records.map(function(r) { return r.bat != null ? r.bat : null; });
    segments = records.map(function(r) { return zoneColors[r.zone || classifyZone(r.avg, r.peak, r.tilt, r.lidAngle, r.lidAngle > 0)] || '#999'; });
    sunMarks = sunMarkersForRange(0, 1440, cursor);

  } else if (gran === 'hour') {
    // Rolling 4-hour window ending at cursor's hour
    const endH = cursor.getHours();
    const startH = endH - 3;
    var startDt = new Date(cursor); startDt.setHours(startH, 0, 0, 0);
    var endDt = new Date(cursor); endDt.setHours(endH, 59, 0, 0);
    navLabel.textContent = dateStr(startDt).slice(5) + ' ' + padH(startDt.getHours()) + ':00 \u2014 ' + padH(endH) + ':59';

    var dayCache = {};
    for (var i = 0; i < 4; i++) {
      var hr = new Date(startDt); hr.setHours(startDt.getHours() + i);
      var d = dateStr(hr);
      if (!dayCache[d]) dayCache[d] = await fetchDay(d);
      var hourStr = padH(hr.getHours());
      var mins = dayCache[d].filter(function(r) { return r.t.startsWith(hourStr + ':'); });
      var avg = mins.length ? mins.reduce(function(s,r) { return s + r.avg; }, 0) / mins.length : 0;
      var pk = mins.length ? mins.reduce(function(s,r) { return s + (r.peak || 0); }, 0) / mins.length : 0;
      var openMins = mins.filter(function(r) { return r.lid !== false; });
      var tl = openMins.length ? openMins.reduce(function(s,r) { return s + (r.tilt || 0); }, 0) / openMins.length : 0;
      var la = openMins.length ? openMins.reduce(function(s,r) { return s + (r.lidAngle || 0); }, 0) / openMins.length : 0;
      var lidFrac = mins.length ? mins.filter(function(r) { return r.lid === false; }).length / mins.length : 0;
      var zone = classifyZone(avg, pk, tl, la, lidFrac < 0.5);
      labels.push(hourStr + ':00');
      data.push(avg);
      peakData.push(pk);
      tiltData.push(tl);
      lidAngleData.push(la);
      // lidData left empty for aggregated views (no hatch)
      segments.push(zoneColors[zone]);
    }
    var sMinH = ((startH % 24) + 24) % 24 * 60;
    var eMinH = ((endH % 24) + 24) % 24 * 60 + 59;
    sunMarks = sunMarkersForRange(sMinH, eMinH > sMinH ? eMinH : eMinH + 1440, cursor);

  } else if (gran === 'daypart') {
    var date = dateStr(cursor);
    navLabel.textContent = date;
    var records = await fetchDay(date);
    dayparts.forEach(function(dp) {
      var dpStart = parseInt(dp.slice(0,2));
      var dpEnd = parseInt(dp.slice(3,5));
      var mins = records.filter(function(r) { var h = parseInt(r.t.slice(0,2)); return h >= dpStart && h < dpEnd; });
      var avg = mins.length ? mins.reduce(function(s,r) { return s + r.avg; }, 0) / mins.length : 0;
      var pk = mins.length ? mins.reduce(function(s,r) { return s + (r.peak || 0); }, 0) / mins.length : 0;
      var openMins = mins.filter(function(r) { return r.lid !== false; });
      var tl = openMins.length ? openMins.reduce(function(s,r) { return s + (r.tilt || 0); }, 0) / openMins.length : 0;
      var la = openMins.length ? openMins.reduce(function(s,r) { return s + (r.lidAngle || 0); }, 0) / openMins.length : 0;
      var lidFrac = mins.length ? mins.filter(function(r) { return r.lid === false; }).length / mins.length : 0;
      var zone = classifyZone(avg, pk, tl, la, lidFrac < 0.5);
      labels.push(dp);
      data.push(avg);
      peakData.push(pk);
      tiltData.push(tl);
      lidAngleData.push(la);
      // lidData left empty for aggregated views (no hatch)
      segments.push(zoneColors[zone]);
    });
    sunMarks = sunMarkersForRange(0, 1440, cursor);

  } else if (gran === 'day') {
    var weekEnd = new Date(cursor);
    var weekStart = addDays(weekEnd, -6);
    navLabel.textContent = dateStr(weekStart).slice(5) + ' to ' + dateStr(weekEnd).slice(5);
    var days = await fetchRange(dateStr(weekStart), dateStr(weekEnd));
    var dayNames = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
    for (var i = 0; i < 7; i++) {
      var d = addDays(weekStart, i);
      var ds = dateStr(d);
      var dayData = days.find(function(dy) { return dy.date === ds; });
      var recs = dayData ? dayData.records : [];
      var avg = recs.length ? recs.reduce(function(s,r) { return s + r.avg; }, 0) / recs.length : 0;
      var pk = recs.length ? recs.reduce(function(s,r) { return s + (r.peak || 0); }, 0) / recs.length : 0;
      var openRecs = recs.filter(function(r) { return r.lid !== false; });
      var tl = openRecs.length ? openRecs.reduce(function(s,r) { return s + (r.tilt || 0); }, 0) / openRecs.length : 0;
      var la = openRecs.length ? openRecs.reduce(function(s,r) { return s + (r.lidAngle || 0); }, 0) / openRecs.length : 0;
      var lidFrac = recs.length ? recs.filter(function(r) { return r.lid === false; }).length / recs.length : 0;
      var zone = classifyZone(avg, pk, tl, la, lidFrac < 0.5);
      labels.push(dayNames[d.getDay()] + ' ' + ds.slice(5));
      data.push(avg);
      peakData.push(pk);
      tiltData.push(tl);
      // lidData left empty for aggregated views (no hatch)
      lidAngleData.push(la);
      segments.push(zoneColors[zone]);
    }

  } else if (gran === 'week') {
    var monthStart = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
    var monthEnd = new Date(cursor.getFullYear(), cursor.getMonth()+1, 0);
    var monthNames = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    navLabel.textContent = monthNames[cursor.getMonth()] + ' ' + cursor.getFullYear();
    var days = await fetchRange(dateStr(monthStart), dateStr(monthEnd));
    var wStart = new Date(monthStart);
    while (wStart <= monthEnd) {
      var wEnd = addDays(wStart, 6 - ((wStart.getDay()+6)%7));
      if (wEnd > monthEnd) wEnd = monthEnd;
      var cw = getISOWeek(wStart);
      var allRecs = [];
      days.forEach(function(dy) {
        if (dy.date >= dateStr(wStart) && dy.date <= dateStr(wEnd)) {
          allRecs.push.apply(allRecs, dy.records);
        }
      });
      var avg = allRecs.length ? allRecs.reduce(function(s,r) { return s + r.avg; }, 0) / allRecs.length : 0;
      var pk = allRecs.length ? allRecs.reduce(function(s,r) { return s + (r.peak || 0); }, 0) / allRecs.length : 0;
      var openAll = allRecs.filter(function(r) { return r.lid !== false; });
      var tl = openAll.length ? openAll.reduce(function(s,r) { return s + (r.tilt || 0); }, 0) / openAll.length : 0;
      var la = openAll.length ? openAll.reduce(function(s,r) { return s + (r.lidAngle || 0); }, 0) / openAll.length : 0;
      var lidFrac = allRecs.length ? allRecs.filter(function(r) { return r.lid === false; }).length / allRecs.length : 0;
      var zone = classifyZone(avg, pk, tl, la, lidFrac < 0.5);
      labels.push('CW' + cw);
      data.push(avg);
      peakData.push(pk);
      tiltData.push(tl);
      lidAngleData.push(la);
      // lidData left empty for aggregated views (no hatch)
      segments.push(zoneColors[zone]);
      wStart = addDays(wEnd, 1);
    }
  }

  renderChart(labels, data, peakData, lidData, tiltData, lidAngleData, acData, batData, segments, sunMarks);
}

function labelToMin(lbl) {
  if (lbl.indexOf('-') > 0 && lbl.length <= 5) {
    var s = parseInt(lbl.slice(0,2)), e = parseInt(lbl.slice(3,5));
    return (s + e) / 2 * 60;
  }
  if (lbl.indexOf(':') > 0) return parseInt(lbl.slice(0,2)) * 60 + parseInt(lbl.slice(3,5));
  return -1;
}

function renderChart(labels, data, peakData, lidData, tiltData, lidAngleData, acData, batData, segColors, sunMarks) {
  if (chart) chart.destroy();
  var ctx = document.getElementById('chart');
  var labelMins = labels.map(labelToMin);
  // Fix cross-midnight monotonicity
  for (var i = 1; i < labelMins.length; i++) {
    if (labelMins[i] >= 0 && labelMins[i] < labelMins[i-1]) {
      for (var j = i; j < labelMins.length; j++) { if (labelMins[j] >= 0) labelMins[j] += 1440; }
      break;
    }
  }

  // Create a diagonal hatch pattern for lid-closed regions
  var hatchPattern = (function() {
    var pc = document.createElement('canvas'); pc.width = 8; pc.height = 8;
    var px = pc.getContext('2d');
    px.strokeStyle = '#88888866'; px.lineWidth = 1;
    px.beginPath(); px.moveTo(0, 8); px.lineTo(8, 0); px.stroke();
    return px.createPattern(pc, 'repeat');
  })();

  var zoneBgPlugin = {
    id: 'zoneBg',
    beforeDraw: function(ch) {
      if (!segColors || !segColors.length) return;
      var xScale = ch.scales.x, yScale = ch.scales.y, c = ch.ctx;
      var h = yScale.bottom - yScale.top;
      for (var i = 0; i < segColors.length; i++) {
        var x0 = i === 0 ? xScale.left : (xScale.getPixelForValue(i-1) + xScale.getPixelForValue(i)) / 2;
        var x1 = i === segColors.length-1 ? xScale.right : (xScale.getPixelForValue(i) + xScale.getPixelForValue(i+1)) / 2;
        var lidVal = lidData ? lidData[i] : null;
        var closed = lidVal === false || (typeof lidVal === 'number' && lidVal > 0);
        c.fillStyle = segColors[i] + '15';
        c.fillRect(x0, yScale.top, x1 - x0, h);
        if (closed && hatchPattern) {
          c.save();
          c.globalAlpha = lidVal === false ? 1 : lidVal;
          c.fillStyle = hatchPattern;
          c.fillRect(x0, yScale.top, x1 - x0, h);
          c.restore();
        }
      }
    }
  };

  var lidPlugin = {
    id: 'lidBar',
    afterDraw: function(ch) {
      if (!lidData || !lidData.length) return;
      var xScale = ch.scales.x, yScale = ch.scales.y, c = ch.ctx;
      var barH = 3, barY = yScale.top - 1;
      for (var i = 0; i < lidData.length; i++) {
        var lv = lidData[i];
        var isClosed = lv === false || (typeof lv === 'number' && lv > 0);
        if (!isClosed) continue;
        var x0 = i === 0 ? xScale.left : (xScale.getPixelForValue(i-1) + xScale.getPixelForValue(i)) / 2;
        var x1 = i === lidData.length-1 ? xScale.right : (xScale.getPixelForValue(i) + xScale.getPixelForValue(i+1)) / 2;
        var alpha = lv === false ? 0.4 : lv * 0.4;
        c.fillStyle = 'rgba(136,136,136,' + alpha + ')';
        c.fillRect(x0, barY, x1 - x0, barH);
      }
    }
  };

  var sunPlugin = {
    id: 'sunLines',
    afterDraw: function(ch) {
      if (!sunMarks || !sunMarks.length || labelMins[0] < 0) return;
      var xScale = ch.scales.x, yScale = ch.scales.y, c = ch.ctx;
      sunMarks.forEach(function(m) {
        var sm = m.min;
        if (labelMins[0] > 1000 && sm < 500) sm += 1440;
        if (sm < labelMins[0] || sm > labelMins[labelMins.length-1]) return;
        var xPos = -1;
        for (var i = 0; i < labelMins.length - 1; i++) {
          if (sm >= labelMins[i] && sm <= labelMins[i+1]) {
            var frac = (sm - labelMins[i]) / (labelMins[i+1] - labelMins[i]);
            var p0 = xScale.getPixelForValue(i), p1 = xScale.getPixelForValue(i+1);
            xPos = p0 + frac * (p1 - p0);
            break;
          }
        }
        if (xPos < 0) return;
        c.save();
        c.strokeStyle = m.color;
        c.lineWidth = 1;
        c.setLineDash([3, 3]);
        c.beginPath();
        c.moveTo(xPos, yScale.top);
        c.lineTo(xPos, yScale.bottom);
        c.stroke();
        c.setLineDash([]);
        c.fillStyle = m.color;
        c.font = '10px -apple-system, sans-serif';
        c.textAlign = 'center';
        c.fillText(m.icon + ' ' + m.label, xPos, yScale.top - 4);
        c.restore();
      });
    }
  };

  var acMarks = [];
  if (acData && acData.length > 1) {
    for (var ai = 1; ai < acData.length; ai++) {
      if (acData[ai] === true && acData[ai-1] === false) {
        acMarks.push({idx: ai, icon: '\u25b2', color: '#00cc66'});
      } else if (acData[ai] === false && acData[ai-1] === true) {
        acMarks.push({idx: ai, icon: '\u25bc', color: '#ff4444'});
      }
    }
  }

  var acPlugin = {
    id: 'acLines',
    afterDraw: function(ch) {
      if (!acMarks || !acMarks.length) return;
      var xScale = ch.scales.x, yScale = ch.scales.y, c = ch.ctx;
      acMarks.forEach(function(m) {
        var xPos = xScale.getPixelForValue(m.idx);
        if (xPos < xScale.left || xPos > xScale.right) return;
        c.save();
        c.strokeStyle = m.color;
        c.lineWidth = 1;
        c.setLineDash([3, 3]);
        c.beginPath();
        c.moveTo(xPos, yScale.top);
        c.lineTo(xPos, yScale.bottom);
        c.stroke();
        c.setLineDash([]);
        c.fillStyle = m.color;
        c.font = '11px -apple-system, sans-serif';
        c.textAlign = 'center';
        c.fillText(m.icon, xPos, yScale.top - 4);
        c.restore();
      });
    }
  };

  var hasPeak = peakData && peakData.some(function(v) { return v > 0; });
  var datasets = [{
    label: 'Average',
    data: data,
    borderColor: '#888',
    backgroundColor: '#88888811',
    fill: true,
    pointRadius: 3,
    pointBackgroundColor: segColors,
    pointBorderColor: segColors,
    borderWidth: 1.5,
    tension: 0.3,
    yAxisID: 'y'
  }];

  if (hasPeak) {
    datasets.push({
      label: 'Peak',
      data: peakData,
      borderColor: '#ff6600',
      backgroundColor: 'transparent',
      fill: false,
      pointRadius: 0,
      borderWidth: 1,
      borderDash: [6, 3],
      tension: 0.3,
      yAxisID: 'y1'
    });
  }

  var hasBat = batData && batData.some(function(v) { return v != null && v >= 0; });
  if (hasBat) {
    datasets.push({
      label: 'Battery',
      data: batData,
      borderColor: '#ff4444',
      segment: { borderColor: function(ctx) { return (acData && acData[ctx.p1DataIndex] === true) ? '#00cc66' : '#ff4444'; } },
      backgroundColor: 'transparent',
      fill: false,
      pointRadius: 0,
      borderWidth: 1,
      borderDash: [2, 3],
      tension: 0.3,
      yAxisID: 'y2'
    });
  }

  var scales = {
    x: { ticks: { color: getCSS('--dim') }, grid: { color: getCSS('--gridline') } },
    y: { ticks: { color: getCSS('--dim') }, grid: { color: getCSS('--gridline') }, title: { display: true, text: 'avg (g)', color: getCSS('--dim') }, beginAtZero: true, position: 'left' }
  };
  if (hasPeak) {
    var avgMax = Math.max.apply(null, data);
    var peakMax = Math.max.apply(null, peakData);
    var sharedMax = Math.max(avgMax, peakMax) * 1.1;
    scales.y.suggestedMax = sharedMax;
    scales.y1 = { ticks: { color: '#ff660088' }, grid: { drawOnChartArea: false }, title: { display: true, text: 'peak (g)', color: '#ff660088' }, beginAtZero: true, position: 'right', suggestedMax: sharedMax };
  }
  if (hasBat) {
    scales.y2 = { ticks: { color: '#ff444466', callback: function(v) { return v + '%'; } }, grid: { drawOnChartArea: false }, min: 0, max: 100, display: false };
  }

  chart = new Chart(ctx, {
    type: 'line',
    data: { labels: labels, datasets: datasets },
    options: {
      responsive: true,
      layout: { padding: { top: 16 } },
      plugins: { legend: { display: hasPeak || hasBat, labels: { color: getCSS('--text'), font: { size: 12 }, usePointStyle: true, pointStyleWidth: 40, generateLabels: function(chart) {
        var avgIcon = document.createElement('canvas'); avgIcon.width = 40; avgIcon.height = 14;
        var ac = avgIcon.getContext('2d'); ac.strokeStyle = '#888'; ac.lineWidth = 2;
        ac.beginPath(); ac.moveTo(0, 7); ac.lineTo(40, 7); ac.stroke();
        ac.fillStyle = '#888'; ac.beginPath(); ac.arc(20, 7, 3, 0, Math.PI * 2); ac.fill();
        var pkIcon = document.createElement('canvas'); pkIcon.width = 40; pkIcon.height = 14;
        var pc = pkIcon.getContext('2d'); pc.strokeStyle = '#ff6600'; pc.lineWidth = 2;
        pc.setLineDash([6, 3]); pc.beginPath(); pc.moveTo(0, 7); pc.lineTo(40, 7); pc.stroke();
        var batIcon = document.createElement('canvas'); batIcon.width = 40; batIcon.height = 14;
        var bc = batIcon.getContext('2d'); bc.strokeStyle = '#ff4444'; bc.lineWidth = 1;
        bc.setLineDash([2, 3]); bc.beginPath(); bc.moveTo(0, 7); bc.lineTo(40, 7); bc.stroke();
        var lidIcon = document.createElement('canvas'); lidIcon.width = 40; lidIcon.height = 14;
        var lc = lidIcon.getContext('2d');
        lc.fillStyle = '#88888822'; lc.fillRect(0, 0, 40, 14);
        lc.strokeStyle = '#88888866'; lc.lineWidth = 1;
        for (var k = -14; k < 40; k += 6) { lc.beginPath(); lc.moveTo(k, 14); lc.lineTo(k + 14, 0); lc.stroke(); }
        var dsIcons = [avgIcon, pkIcon, batIcon];
        var items = chart.data.datasets.map(function(ds, i) { return { text: ds.label, fontColor: getCSS('--text'), strokeStyle: 'transparent', fillStyle: 'transparent', pointStyle: dsIcons[i] || avgIcon, hidden: !chart.isDatasetVisible(i), datasetIndex: i }; });
        if (lidData && lidData.some(function(v) { return v === false || (typeof v === 'number' && v > 0); })) {
          items.push({ text: 'Lid closed', fontColor: getCSS('--text'), strokeStyle: 'transparent', fillStyle: 'transparent', pointStyle: lidIcon, hidden: false });
        }
        return items;
      } } },
      tooltip: { callbacks: { title: function(ctx) { return ctx[0].label; }, label: function(ctx) {
        var i = ctx.dataIndex;
        if (ctx.datasetIndex === 1) return 'Peak: ' + ctx.parsed.y.toFixed(4) + 'g';
        if (ctx.datasetIndex === 2) return 'Battery: ' + ctx.parsed.y + '%';
        var lines = ['Avg: ' + ctx.parsed.y.toFixed(4) + 'g'];
        if (peakData && peakData[i]) lines.push('Peak: ' + peakData[i].toFixed(4) + 'g');
        if (tiltData && tiltData[i] != null) lines.push('Tilt: ' + tiltData[i].toFixed(1) + '\u00b0');
        if (lidAngleData && lidAngleData[i]) lines.push('Lid angle: ' + lidAngleData[i].toFixed(0) + '\u00b0');
        if (lidData && lidData.length) {
          var lv = lidData[i];
          if (lv === false) lines.push('Lid: closed');
          else if (lv === true) lines.push('Lid: open');
        }
        if (acData && acData.length) {
          var acOn = acData[i] === true;
          var batPct = (batData && batData[i] != null && batData[i] >= 0) ? batData[i] + '%' : '';
          if (acOn && batPct) lines.push('AC connected, ' + (batData[i] < 100 ? 'charging ' : '') + batPct);
          else if (acOn) lines.push('AC connected');
          else if (batPct) lines.push('Battery ' + batPct);
          else lines.push('Battery');
        }
        return lines;
      } } } },
      scales: scales
    },
    plugins: [zoneBgPlugin, lidPlugin, sunPlugin, acPlugin]
  });
}

// --- Settings ---
function setSettingsTab(tab) {
  ['general','telegram','email','alarm','about'].forEach(function(t) {
    document.getElementById('tab-content-' + t).style.display = t === tab ? 'block' : 'none';
    var btn = document.getElementById('tab-' + t);
    btn.style.borderBottomColor = t === tab ? 'var(--accent)' : 'transparent';
    btn.style.color = t === tab ? 'var(--text)' : 'var(--muted)';
  });
}

function toggleTelegramFields() {
  document.getElementById('telegram-fields').style.display = document.getElementById('set-telegram').checked ? 'block' : 'none';
}

function toggleEmailFields() {
  document.getElementById('email-fields').style.display = document.getElementById('set-email').checked ? 'block' : 'none';
}

async function testAlarmSound(selectId) {
  var sound = document.getElementById(selectId).value;
  try { await fetch('/alarm/test?sound=' + encodeURIComponent(sound), { method: 'POST' }); } catch(e) {}
}

async function openSettings() {
  try {
    const r = await fetch('/settings');
    const s = await r.json();
    document.getElementById('set-delay').value = s.defaultDelay || 0;
    document.getElementById('set-telegram').checked = s.notifyTelegram;
    document.getElementById('set-email').checked = s.notifyEmail;
    document.getElementById('set-chat-id').value = s.telegramChatId || '';
    document.getElementById('set-email-addr').value = s.emailAddress || '';
    document.getElementById('set-smtp-host').value = s.smtpHost || '';
    document.getElementById('set-smtp-user').value = s.smtpUser || '';
    document.getElementById('set-smtp-pass').value = '';
    document.getElementById('set-alarm-enabled').checked = s.alarmEnabled || false;
    document.getElementById('set-alarm-sound').value = s.alarmSound || 'Siren';
    document.getElementById('set-geo-alarm-sound').value = s.geoAlarmSound || 'Intruder';
    document.getElementById('set-ac-alarm').checked = s.acDisconnectAlarm || false;
    document.getElementById('set-ac-alarm-sound').value = s.acAlarmSound || 'Funk';
    var calBtn = document.querySelector('#settings-overlay #calib-btn');
    if (calBtn) calBtn.textContent = 'Calibrate';
    var bi = document.querySelector('#settings-overlay #baseline-info');
    if (bi) { bi.textContent = s.baseline > 0 ? 'Calibrated' : 'Not calibrated'; bi.style.color = s.baseline > 0 ? 'var(--accent)' : 'var(--dim)'; }
    showCalibDetails();
  } catch(e) {}
  setSettingsTab('general');
  toggleTelegramFields();
  toggleEmailFields();
  document.getElementById('settings-overlay').style.display = 'flex';
  document.getElementById('settings-msg').textContent = '';
}

function closeSettings() {
  document.getElementById('settings-overlay').style.display = 'none';
}

async function saveSettingsUI() {
  const s = {
    defaultDelay: parseInt(document.getElementById('set-delay').value) || 0,
    notifyTelegram: document.getElementById('set-telegram').checked,
    notifyEmail: document.getElementById('set-email').checked,
    telegramChatId: parseInt(document.getElementById('set-chat-id').value) || 0,
    emailAddress: document.getElementById('set-email-addr').value,
    smtpHost: document.getElementById('set-smtp-host').value,
    smtpUser: document.getElementById('set-smtp-user').value,
    smtpPass: document.getElementById('set-smtp-pass').value,
    alarmEnabled: document.getElementById('set-alarm-enabled').checked,
    alarmSound: document.getElementById('set-alarm-sound').value,
    geoAlarmSound: document.getElementById('set-geo-alarm-sound').value,
    acAlarmSound: document.getElementById('set-ac-alarm-sound').value,
    acDisconnectAlarm: document.getElementById('set-ac-alarm').checked
  };
  try {
    await fetch('/settings', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(s) });
    document.getElementById('settings-msg').textContent = 'Saved';
    setTimeout(function() { document.getElementById('settings-msg').textContent = ''; }, 2000);
    loadStatus();
  } catch(e) {
    document.getElementById('settings-msg').textContent = 'Error saving';
  }
}

// Close settings on overlay click
document.getElementById('settings-overlay').addEventListener('click', function(e) {
  if (e.target === this) closeSettings();
});

// --- Training ---
let trainingActive = false;
let trainingStart = null;
let trainingTimer = null;

async function toggleTraining() {
  if (trainingActive) {
    const r = await fetch('/training/stop', { method: 'POST' });
    const d = await r.json();
    trainingActive = false;
    clearInterval(trainingTimer);
    document.getElementById('rec-dot').style.display = 'none';
    document.getElementById('rec-label').textContent = 'Rec';
    document.getElementById('rec-btn').style.borderColor = 'var(--picker-border)';
    if (d.records > 0) {
      document.getElementById('rec-label').textContent = d.records + ' records saved';
      setTimeout(function() { document.getElementById('rec-label').textContent = 'Rec'; }, 3000);
    }
  } else {
    const r = await fetch('/training/start', { method: 'POST' });
    const d = await r.json();
    if (d.status === 'recording' || d.status === 'already_recording') {
      trainingActive = true;
      trainingStart = Date.now();
      document.getElementById('rec-dot').style.display = 'block';
      document.getElementById('rec-btn').style.borderColor = '#ff3300';
      updateRecTimer();
      trainingTimer = setInterval(updateRecTimer, 1000);
    }
  }
}

function updateRecTimer() {
  var elapsed = Math.floor((Date.now() - trainingStart) / 1000);
  var m = Math.floor(elapsed / 60);
  var s = elapsed % 60;
  document.getElementById('rec-label').textContent = (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
}

// Check if training is already active on page load
async function checkTrainingStatus() {
  try {
    const r = await fetch('/training/status');
    const d = await r.json();
    if (d.recording) {
      trainingActive = true;
      trainingStart = Date.now();
      document.getElementById('rec-dot').style.display = 'block';
      document.getElementById('rec-btn').style.borderColor = '#ff3300';
      updateRecTimer();
      trainingTimer = setInterval(updateRecTimer, 1000);
    }
  } catch(e) {}
}

// --- Init ---
map = L.map('map', { zoomControl: true, scrollWheelZoom: false }).setView([48.22, 9.01], 13);
document.getElementById('map').addEventListener('wheel', function(e) {
  if (e.ctrlKey || e.metaKey) { map.scrollWheelZoom.enable(); setTimeout(function() { map.scrollWheelZoom.disable(); }, 500); }
}, { passive: true });
const initTile = isDark ? tiles.dark : tiles.light;
tileLayer = L.tileLayer(initTile.url, { attribution: initTile.attr, maxZoom: 19 }).addTo(map);
applyTheme();

// Draw hatched lid-closed icon in zone legend
(function() {
  var c = document.getElementById('zone-lid-icon');
  if (!c) return;
  var x = c.getContext('2d');
  x.fillStyle = '#88888822'; x.fillRect(0, 0, 12, 12);
  x.strokeStyle = '#88888866'; x.lineWidth = 1;
  for (var k = -12; k < 12; k += 4) { x.beginPath(); x.moveTo(k, 12); x.lineTo(k + 12, 0); x.stroke(); }
})();

loadStatus();
loadChart();
refreshLocation();
checkTrainingStatus();
// Load default delay from settings
fetch('/settings').then(r => r.json()).catch(function(){});
setInterval(loadStatus, 2000);
setInterval(loadChart, 60000);
</script>
</body>
</html>`
