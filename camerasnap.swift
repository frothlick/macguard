import AVFoundation
import AppKit
import Foundation

// Usage: camerasnap <output_dir> [count] [interval_ms]
// Captures `count` photos at `interval_ms` intervals + records audio for the duration
// Outputs one JSON line per capture to stdout
// Audio saved as <output_dir>/audio.m4a

class CameraSnapper: NSObject, AVCapturePhotoCaptureDelegate {
    let session = AVCaptureSession()
    let photoOutput = AVCapturePhotoOutput()
    var audioWriter: AVAssetWriter?
    var audioWriterInput: AVAssetWriterInput?
    var audioOutput: AVCaptureAudioDataOutput?
    var audioQueue = DispatchQueue(label: "audio-queue")
    var pendingPath: String = ""
    var captureComplete = false
    var lastStatus: String = ""
    var audioPath: String = ""
    var audioStarted = false

    func setup(dir: String) -> Bool {
        session.sessionPreset = .photo

        // Camera
        guard let videoDevice = AVCaptureDevice.default(for: .video) else {
            print("{\"status\":\"no_camera\"}")
            return false
        }

        let videoAuth = AVCaptureDevice.authorizationStatus(for: .video)
        if videoAuth == .denied || videoAuth == .restricted {
            print("{\"status\":\"camera_denied\"}")
            return false
        }
        if videoAuth == .notDetermined {
            var granted = false
            let sema = DispatchSemaphore(value: 0)
            AVCaptureDevice.requestAccess(for: .video) { g in granted = g; sema.signal() }
            sema.wait()
            if !granted {
                print("{\"status\":\"camera_denied\"}")
                return false
            }
        }

        guard let videoInput = try? AVCaptureDeviceInput(device: videoDevice) else {
            print("{\"status\":\"video_input_error\"}")
            return false
        }
        session.addInput(videoInput)
        session.addOutput(photoOutput)

        // Microphone
        let audioAuth = AVCaptureDevice.authorizationStatus(for: .audio)
        if audioAuth == .notDetermined {
            let sema = DispatchSemaphore(value: 0)
            AVCaptureDevice.requestAccess(for: .audio) { _ in sema.signal() }
            sema.wait()
        }

        if let audioDevice = AVCaptureDevice.default(for: .audio),
           let audioInput = try? AVCaptureDeviceInput(device: audioDevice) {
            session.addInput(audioInput)

            audioPath = (dir as NSString).appendingPathComponent("audio.m4a")
            let audioURL = URL(fileURLWithPath: audioPath)
            try? FileManager.default.removeItem(at: audioURL)

            if let writer = try? AVAssetWriter(outputURL: audioURL, fileType: .m4a) {
                let settings: [String: Any] = [
                    AVFormatIDKey: kAudioFormatMPEG4AAC,
                    AVSampleRateKey: 44100,
                    AVNumberOfChannelsKey: 1,
                    AVEncoderBitRateKey: 128000
                ]
                let writerInput = AVAssetWriterInput(mediaType: .audio, outputSettings: settings)
                writerInput.expectsMediaDataInRealTime = true
                writer.add(writerInput)

                let dataOutput = AVCaptureAudioDataOutput()
                dataOutput.setSampleBufferDelegate(self, queue: audioQueue)
                session.addOutput(dataOutput)

                audioWriter = writer
                audioWriterInput = writerInput
                audioOutput = dataOutput
                writer.startWriting()
                writer.startSession(atSourceTime: .zero)
            } else {
                fputs("warning: could not create audio writer\n", stderr)
            }
        } else {
            fputs("warning: microphone not available\n", stderr)
        }

        session.startRunning()

        // Wait for camera to fully initialize
        let warmup = Date(timeIntervalSinceNow: 2.0)
        while Date() < warmup {
            RunLoop.current.run(mode: .default, before: Date(timeIntervalSinceNow: 0.05))
        }
        return true
    }

    func snap(path: String) -> Bool {
        pendingPath = path
        lastStatus = ""
        captureComplete = false
        let settings = AVCapturePhotoSettings(format: [AVVideoCodecKey: AVVideoCodecType.jpeg])
        photoOutput.capturePhoto(with: settings, delegate: self)

        let deadline = Date(timeIntervalSinceNow: 10)
        while !captureComplete && Date() < deadline {
            RunLoop.current.run(mode: .default, before: Date(timeIntervalSinceNow: 0.05))
        }
        if !captureComplete {
            print("{\"status\":\"timeout\"}")
            return false
        }
        return lastStatus == "ok"
    }

    func stopAudio() {
        guard let writer = audioWriter, let input = audioWriterInput else { return }
        audioQueue.sync {
            input.markAsFinished()
        }
        let sema = DispatchSemaphore(value: 0)
        writer.finishWriting { sema.signal() }
        sema.wait()
        if writer.status == .completed {
            print("{\"status\":\"audio_ok\",\"path\":\"\(audioPath)\"}")
        } else {
            print("{\"status\":\"audio_error\",\"message\":\"\(writer.error?.localizedDescription ?? "unknown")\"}")
        }
    }

    func shutdown() {
        stopAudio()
        session.stopRunning()
    }

    // MARK: - Photo delegate
    func photoOutput(_ output: AVCapturePhotoOutput, didFinishProcessingPhoto photo: AVCapturePhoto, error: Error?) {
        defer { captureComplete = true }

        if let error = error {
            lastStatus = "error"
            print("{\"status\":\"error\",\"message\":\"\(error.localizedDescription)\"}")
            return
        }

        guard let data = photo.fileDataRepresentation() else {
            lastStatus = "no_data"
            print("{\"status\":\"no_data\"}")
            return
        }

        do {
            try data.write(to: URL(fileURLWithPath: pendingPath))
            lastStatus = "ok"
            print("{\"status\":\"ok\",\"path\":\"\(pendingPath)\"}")
        } catch {
            lastStatus = "write_error"
            print("{\"status\":\"write_error\",\"message\":\"\(error.localizedDescription)\"}")
        }
    }
}

// MARK: - Audio delegate
extension CameraSnapper: AVCaptureAudioDataOutputSampleBufferDelegate {
    func captureOutput(_ output: AVCaptureOutput, didOutput sampleBuffer: CMSampleBuffer, from connection: AVCaptureConnection) {
        guard let input = audioWriterInput, let writer = audioWriter else { return }
        if !audioStarted {
            let ts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer)
            writer.startSession(atSourceTime: ts)
            audioStarted = true
        }
        if input.isReadyForMoreMediaData {
            input.append(sampleBuffer)
        }
    }
}

// Main
let args = CommandLine.arguments
let outputDir = args.count > 1 ? args[1] : "/tmp"
let count = args.count > 2 ? Int(args[2]) ?? 5 : 5
let intervalMs = args.count > 3 ? Int(args[3]) ?? 1000 : 1000

try? FileManager.default.createDirectory(atPath: outputDir, withIntermediateDirectories: true)

let snapper = CameraSnapper()
guard snapper.setup(dir: outputDir) else { exit(1) }

for i in 0..<count {
    let path = (outputDir as NSString).appendingPathComponent("capture_\(i).jpg")
    _ = snapper.snap(path: path)
    fflush(stdout)
    if i < count - 1 {
        let sleepUntil = Date(timeIntervalSinceNow: Double(intervalMs) / 1000.0)
        while Date() < sleepUntil {
            RunLoop.current.run(mode: .default, before: Date(timeIntervalSinceNow: 0.05))
        }
    }
}

snapper.shutdown()
fflush(stdout)
