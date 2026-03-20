import CoreLocation
import Foundation

class Locator: NSObject, CLLocationManagerDelegate {
    let manager = CLLocationManager()
    var done = false
    var authorized = false

    func run() {
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyBest

        let status = manager.authorizationStatus
        fputs("initial auth: \(status.rawValue)\n", stderr)

        if status == .notDetermined {
            manager.requestAlwaysAuthorization()
            // Wait for authorization callback
            let authDeadline = Date(timeIntervalSinceNow: 30)
            while !authorized && !done && Date() < authDeadline {
                RunLoop.current.run(mode: .default, before: Date(timeIntervalSinceNow: 0.1))
            }
        } else if status == .authorizedAlways {
            authorized = true
        }

        if !authorized {
            if !done {
                print("{\"status\":\"denied\"}")
            }
            return
        }

        manager.startUpdatingLocation()

        let deadline = Date(timeIntervalSinceNow: 15)
        while !done && Date() < deadline {
            RunLoop.current.run(mode: .default, before: Date(timeIntervalSinceNow: 0.1))
        }

        if !done {
            print("{\"status\":\"timeout\"}")
        }
    }

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        let status = manager.authorizationStatus
        fputs("auth changed: \(status.rawValue)\n", stderr)
        if status == .authorizedAlways {
            authorized = true
        } else if status == .denied || status == .restricted {
            done = true
            print("{\"status\":\"denied\"}")
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard !done, let loc = locations.last else { return }
        if loc.horizontalAccuracy >= 0 && loc.horizontalAccuracy < 500 {
            done = true
            manager.stopUpdatingLocation()
            print("{\"lat\":\(loc.coordinate.latitude),\"lon\":\(loc.coordinate.longitude),\"acc\":\(loc.horizontalAccuracy),\"status\":\"ok\"}")
        }
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        fputs("location error: \(error)\n", stderr)
        done = true
        print("{\"status\":\"unavailable\"}")
    }
}

let locator = Locator()
locator.run()
