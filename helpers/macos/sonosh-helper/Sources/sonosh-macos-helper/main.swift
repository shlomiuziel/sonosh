import Darwin
import AppKit
import ApplicationServices
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
            MediaKeyHUD.shared.show(command: "play")
            connection.send(.command("play"))
            return .success
        }

        center.pauseCommand.isEnabled = true
        center.pauseCommand.addTarget { [connection] _ in
            MediaKeyHUD.shared.show(command: "pause")
            connection.send(.command("pause"))
            return .success
        }

        center.togglePlayPauseCommand.isEnabled = true
        center.togglePlayPauseCommand.addTarget { [connection] _ in
            MediaKeyHUD.shared.show(command: "togglePlayPause")
            connection.send(.command("togglePlayPause"))
            return .success
        }

        center.nextTrackCommand.isEnabled = true
        center.nextTrackCommand.addTarget { [connection] _ in
            MediaKeyHUD.shared.show(command: "next")
            connection.send(.command("next"))
            return .success
        }

        center.previousTrackCommand.isEnabled = true
        center.previousTrackCommand.addTarget { [connection] _ in
            MediaKeyHUD.shared.show(command: "previous")
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
        case "settings":
            if let enabled = message.hudEnabled {
                MediaKeyHUD.shared.setEnabled(enabled)
            }
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

final class MediaKeyHUD {
    static let shared = MediaKeyHUD()

    private var panel: NSPanel?
    private var symbolView: NSImageView?
    private var titleField: NSTextField?
    private var hideWorkItem: DispatchWorkItem?
    private var isEnabled = true

    private init() {}

    private func ensurePanel() {
        if panel != nil {
            return
        }

        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 240, height: 120),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.isFloatingPanel = true
        panel.level = .statusBar
        panel.backgroundColor = .clear
        panel.isOpaque = false
        panel.hasShadow = true
        panel.hidesOnDeactivate = false
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .transient]
        panel.ignoresMouseEvents = true
        panel.alphaValue = 0

        let visualEffectView = NSVisualEffectView(frame: panel.contentView?.bounds ?? .zero)
        visualEffectView.translatesAutoresizingMaskIntoConstraints = false
        visualEffectView.material = .hudWindow
        visualEffectView.blendingMode = .behindWindow
        visualEffectView.state = .active
        visualEffectView.wantsLayer = true
        visualEffectView.layer?.cornerRadius = 20
        visualEffectView.layer?.masksToBounds = true
        panel.contentView = visualEffectView

        let symbolView = NSImageView()
        symbolView.translatesAutoresizingMaskIntoConstraints = false
        symbolView.imageScaling = .scaleProportionallyUpOrDown
        symbolView.contentTintColor = .white

        let titleField = NSTextField(labelWithString: "")
        titleField.translatesAutoresizingMaskIntoConstraints = false
        titleField.font = .systemFont(ofSize: 18, weight: .semibold)
        titleField.textColor = .white
        titleField.alignment = .center
        titleField.maximumNumberOfLines = 2
        titleField.lineBreakMode = .byTruncatingTail

        visualEffectView.addSubview(symbolView)
        visualEffectView.addSubview(titleField)

        NSLayoutConstraint.activate([
            symbolView.centerXAnchor.constraint(equalTo: visualEffectView.centerXAnchor),
            symbolView.topAnchor.constraint(equalTo: visualEffectView.topAnchor, constant: 22),
            symbolView.widthAnchor.constraint(equalToConstant: 34),
            symbolView.heightAnchor.constraint(equalToConstant: 34),

            titleField.leadingAnchor.constraint(equalTo: visualEffectView.leadingAnchor, constant: 16),
            titleField.trailingAnchor.constraint(equalTo: visualEffectView.trailingAnchor, constant: -16),
            titleField.topAnchor.constraint(equalTo: symbolView.bottomAnchor, constant: 12),
            titleField.bottomAnchor.constraint(lessThanOrEqualTo: visualEffectView.bottomAnchor, constant: -18)
        ])

        self.panel = panel
        self.symbolView = symbolView
        self.titleField = titleField
    }

    func show(command: String) {
        DispatchQueue.main.async {
            self.present(command: command)
        }
    }

    func setEnabled(_ enabled: Bool) {
        DispatchQueue.main.async {
            self.isEnabled = enabled
            if !enabled {
                self.hideWorkItem?.cancel()
                self.dismiss()
            }
        }
    }

    private func present(command: String) {
        guard isEnabled else {
            return
        }
        guard let presentation = hudPresentation(for: command) else {
            return
        }
        ensurePanel()
        guard let panel, let symbolView, let titleField else {
            return
        }

        hideWorkItem?.cancel()
        titleField.stringValue = presentation.title
        symbolView.image = presentation.symbolName.flatMap {
            NSImage(systemSymbolName: $0, accessibilityDescription: presentation.title)
        }
        symbolView.isHidden = symbolView.image == nil

        positionPanel()
        panel.orderFrontRegardless()
        NSAnimationContext.runAnimationGroup { context in
            context.duration = 0.12
            panel.animator().alphaValue = 1
        }

        let hide = DispatchWorkItem { [weak self] in
            self?.dismiss()
        }
        hideWorkItem = hide
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.75, execute: hide)
    }

    private func dismiss() {
        guard let panel else {
            return
        }
        NSAnimationContext.runAnimationGroup({ context in
            context.duration = 0.18
            panel.animator().alphaValue = 0
        }, completionHandler: { [panel] in
            panel.orderOut(nil)
        })
    }

    private func positionPanel() {
        guard let panel else {
            return
        }
        let targetFrame = frameForPanel()
        panel.setFrame(targetFrame, display: false)
    }

    private func frameForPanel() -> NSRect {
        let fallback = NSRect(x: 0, y: 0, width: 240, height: 120)
        let screen = NSScreen.main ?? NSScreen.screens.first
        guard let frame = screen?.visibleFrame else {
            return fallback
        }
        let originX = frame.midX - fallback.width / 2
        let originY = frame.minY + 120
        return NSRect(x: originX, y: originY, width: fallback.width, height: fallback.height)
    }
}

private func hudPresentation(for command: String) -> (title: String, symbolName: String?)? {
    switch command {
    case "play":
        return ("Play", "play.fill")
    case "pause":
        return ("Pause", "pause.fill")
    case "togglePlayPause":
        return ("Play/Pause", "playpause.fill")
    case "next":
        return ("Next", "forward.fill")
    case "previous":
        return ("Previous", "backward.fill")
    case "volumeUp":
        return ("Volume Up", "speaker.wave.2.fill")
    case "volumeDown":
        return ("Volume Down", "speaker.wave.1.fill")
    default:
        return nil
    }
}

private let nxSubtypeAuxControlButtons: Int16 = 8
private let nxKeyStateDown = 0x0A
private let nxKeyTypeSoundUp = 0
private let nxKeyTypeSoundDown = 1
private let cgEventTypeSystemDefined = 14

final class VolumeKeyMonitor: @unchecked Sendable {
    private let connection: IPCConnection
    private var eventTap: CFMachPort?
    private var runLoopSource: CFRunLoopSource?

    init(connection: IPCConnection) {
        self.connection = connection
    }

    deinit {
        if let runLoopSource {
            CFRunLoopRemoveSource(CFRunLoopGetMain(), runLoopSource, .commonModes)
        }
        if let eventTap {
            CFMachPortInvalidate(eventTap)
        }
    }

    func install() -> String? {
        guard ensureAccessibilityPermission() else {
            return "volume keys require Accessibility access; grant it in System Settings > Privacy & Security > Accessibility"
        }

        let mask = (1 as CGEventMask) << cgEventTypeSystemDefined
        let callback: CGEventTapCallBack = { _, type, event, userInfo in
            if type == .tapDisabledByTimeout || type == .tapDisabledByUserInput {
                if let userInfo {
                    let monitor = Unmanaged<VolumeKeyMonitor>.fromOpaque(userInfo).takeUnretainedValue()
                    monitor.reenableEventTap()
                }
                return Unmanaged.passUnretained(event)
            }
            guard let userInfo else {
                return Unmanaged.passUnretained(event)
            }
            let monitor = Unmanaged<VolumeKeyMonitor>.fromOpaque(userInfo).takeUnretainedValue()
            return monitor.handle(event: event)
        }

        let userInfo = UnsafeMutableRawPointer(Unmanaged.passUnretained(self).toOpaque())
        guard let eventTap = CGEvent.tapCreate(tap: .cgSessionEventTap, place: .headInsertEventTap, options: .defaultTap, eventsOfInterest: mask, callback: callback, userInfo: userInfo) else {
            return "failed to install volume key event tap; verify Accessibility access is enabled"
        }

        let source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, eventTap, 0)
        self.eventTap = eventTap
        self.runLoopSource = source
        CFRunLoopAddSource(CFRunLoopGetMain(), source, .commonModes)
        CGEvent.tapEnable(tap: eventTap, enable: true)
        return nil
    }

    private func reenableEventTap() {
        guard let eventTap else {
            return
        }
        CGEvent.tapEnable(tap: eventTap, enable: true)
    }

    private func handle(event: CGEvent) -> Unmanaged<CGEvent>? {
        guard let nsEvent = NSEvent(cgEvent: event) else {
            return Unmanaged.passUnretained(event)
        }
        guard let command = volumeCommand(for: nsEvent) else {
            return Unmanaged.passUnretained(event)
        }
        MediaKeyHUD.shared.show(command: command)
        connection.send(.command(command))
        return nil
    }
}

func ensureAccessibilityPermission() -> Bool {
    let options = [kAXTrustedCheckOptionPrompt.takeUnretainedValue() as String: true] as CFDictionary
    return AXIsProcessTrustedWithOptions(options)
}

func volumeCommand(for event: NSEvent) -> String? {
    volumeCommand(subtype: event.subtype.rawValue, data1: Int(event.data1))
}

func volumeCommand(subtype: Int16, data1: Int) -> String? {
    guard subtype == nxSubtypeAuxControlButtons else {
        return nil
    }

    let keyCode = (data1 & 0xFFFF0000) >> 16
    let keyFlags = data1 & 0x0000FFFF
    let keyState = (keyFlags & 0xFF00) >> 8
    let isRepeat = (keyFlags & 0x1) == 0x1
    guard keyState == nxKeyStateDown, !isRepeat else {
        return nil
    }

    switch keyCode {
    case nxKeyTypeSoundUp:
        return "volumeUp"
    case nxKeyTypeSoundDown:
        return "volumeDown"
    default:
        return nil
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
    let volumeKeys = VolumeKeyMonitor(connection: connection)
    bridge.installRemoteCommands()
    if let errorMessage = volumeKeys.install() {
        connection.send(.error(errorMessage))
    }
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
