import Darwin
import AppKit
import Foundation
import MediaPlayer
import SonoshHelperCore

final class IPCConnection: @unchecked Sendable {
    private let fd: Int32
    private let lock = NSLock()

    init(socketPath: String) throws {
        fd = try connectUnixSocket(path: socketPath)
    }

    deinit {
        close(fd)
    }

    func send(_ message: IPCMessage) {
        do {
            let data = try encodeLine(message)
            try data.withUnsafeBytes { rawBuffer in
                guard let base = rawBuffer.baseAddress else { return }
                var sent = 0
                lock.lock()
                defer { lock.unlock() }
                while sent < data.count {
                    let n = Darwin.write(fd, base.advanced(by: sent), data.count - sent)
                    if n <= 0 {
                        throw POSIXError(.EPIPE)
                    }
                    sent += n
                }
            }
        } catch {
            fputs("sonosh-macos-helper: send failed: \(error)\n", stderr)
        }
    }

    func readLoop(onMessage: @Sendable @escaping (IPCMessage) -> Void, onClose: @Sendable @escaping () -> Void) {
        DispatchQueue.global(qos: .userInitiated).async {
            var pending = Data()
            var buffer = [UInt8](repeating: 0, count: 4096)
            while true {
                let n = Darwin.read(self.fd, &buffer, buffer.count)
                if n <= 0 {
                    onClose()
                    return
                }
                pending.append(buffer, count: n)
                while let newline = pending.firstIndex(of: 0x0a) {
                    let line = pending[..<newline]
                    pending.removeSubrange(...newline)
                    if line.isEmpty {
                        continue
                    }
                    do {
                        onMessage(try decodeLine(Data(line)))
                    } catch {
                        self.send(.error("decode: \(error)"))
                    }
                }
            }
        }
    }
}

final class MediaBridge: @unchecked Sendable {
    private let connection: IPCConnection

    init(connection: IPCConnection) {
        self.connection = connection
    }

    func installRemoteCommands() {
        let center = MPRemoteCommandCenter.shared()

        center.playCommand.isEnabled = true
        center.playCommand.addTarget { [connection] _ in
            connection.send(.command("play"))
            return .success
        }

        center.pauseCommand.isEnabled = true
        center.pauseCommand.addTarget { [connection] _ in
            connection.send(.command("pause"))
            return .success
        }

        center.togglePlayPauseCommand.isEnabled = true
        center.togglePlayPauseCommand.addTarget { [connection] _ in
            connection.send(.command("togglePlayPause"))
            return .success
        }

        center.nextTrackCommand.isEnabled = true
        center.nextTrackCommand.addTarget { [connection] _ in
            connection.send(.command("next"))
            return .success
        }

        center.previousTrackCommand.isEnabled = true
        center.previousTrackCommand.addTarget { [connection] _ in
            connection.send(.command("previous"))
            return .success
        }
    }

    func handle(_ message: IPCMessage) {
        switch message.type {
        case "hello":
            if message.protocol != nil && message.protocol != protocolVersion {
                connection.send(.error("unsupported protocol \(message.protocol ?? -1)"))
            }
        case "nowPlaying":
            updateNowPlaying(message)
        case "clear":
            MPNowPlayingInfoCenter.default().nowPlayingInfo = nil
            MPNowPlayingInfoCenter.default().playbackState = .stopped
        default:
            break
        }
    }

    private func updateNowPlaying(_ message: IPCMessage) {
        var info: [String: Any] = [:]
        if let title = nonEmpty(message.title) {
            info[MPMediaItemPropertyTitle] = title
        }
        if let artist = nonEmpty(message.artist) {
            info[MPMediaItemPropertyArtist] = artist
        }
        if let album = nonEmpty(message.album) {
            info[MPMediaItemPropertyAlbumTitle] = album
        }
        if let position = message.positionSeconds {
            info[MPNowPlayingInfoPropertyElapsedPlaybackTime] = position
        }
        if let duration = message.durationSeconds, duration > 0 {
            info[MPMediaItemPropertyPlaybackDuration] = duration
        }
        let playing = message.state == "playing"
        info[MPNowPlayingInfoPropertyPlaybackRate] = playing ? 1.0 : 0.0
        MPNowPlayingInfoCenter.default().nowPlayingInfo = info
        switch message.state {
        case "playing":
            MPNowPlayingInfoCenter.default().playbackState = .playing
        case "paused":
            MPNowPlayingInfoCenter.default().playbackState = .paused
        case "stopped":
            MPNowPlayingInfoCenter.default().playbackState = .stopped
        default:
            MPNowPlayingInfoCenter.default().playbackState = .unknown
        }
    }
}

func connectUnixSocket(path: String) throws -> Int32 {
    let fd = socket(AF_UNIX, SOCK_STREAM, 0)
    if fd < 0 {
        throw POSIXError(POSIXErrorCode(rawValue: errno) ?? .EIO)
    }

    var addr = sockaddr_un()
    addr.sun_family = sa_family_t(AF_UNIX)
    let maxPathLength = MemoryLayout.size(ofValue: addr.sun_path)
    guard path.utf8.count < maxPathLength else {
        close(fd)
        throw POSIXError(.ENAMETOOLONG)
    }

    withUnsafeMutablePointer(to: &addr.sun_path) { pointer in
        pointer.withMemoryRebound(to: CChar.self, capacity: maxPathLength) { buffer in
            _ = path.withCString { source in
                strncpy(buffer, source, maxPathLength - 1)
            }
        }
    }

    let length = socklen_t(MemoryLayout<sockaddr_un>.offset(of: \.sun_path)! + path.utf8.count + 1)
    let result = withUnsafePointer(to: &addr) { pointer in
        pointer.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockaddrPointer in
            Darwin.connect(fd, sockaddrPointer, length)
        }
    }
    if result != 0 {
        let code = POSIXErrorCode(rawValue: errno) ?? .EIO
        close(fd)
        throw POSIXError(code)
    }
    return fd
}

func nonEmpty(_ value: String?) -> String? {
    guard let value else { return nil }
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    return trimmed.isEmpty ? nil : trimmed
}

func socketPath(from arguments: [String]) -> String? {
    for i in arguments.indices {
        if arguments[i] == "--socket", arguments.indices.contains(i + 1) {
            return arguments[i + 1]
        }
    }
    return nil
}

guard let path = socketPath(from: CommandLine.arguments) else {
    fputs("usage: sonosh-macos-helper --socket <path>\n", stderr)
    exit(2)
}

do {
    NSApplication.shared.setActivationPolicy(.accessory)
    NSApplication.shared.finishLaunching()

    let connection = try IPCConnection(socketPath: path)
    let bridge = MediaBridge(connection: connection)
    bridge.installRemoteCommands()
    connection.send(.ready())
    connection.readLoop(
        onMessage: { message in bridge.handle(message) },
        onClose: {
            MPNowPlayingInfoCenter.default().nowPlayingInfo = nil
            MPNowPlayingInfoCenter.default().playbackState = .stopped
            exit(0)
        }
    )
    RunLoop.main.run()
} catch {
    fputs("sonosh-macos-helper: \(error)\n", stderr)
    exit(1)
}
