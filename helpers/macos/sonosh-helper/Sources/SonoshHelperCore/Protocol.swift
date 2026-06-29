import Foundation

public let protocolVersion = 1

public struct IPCMessage: Codable, Equatable {
    public var type: String
    public var `protocol`: Int?
    public var room: String?
    public var state: String?
    public var title: String?
    public var artist: String?
    public var album: String?
    public var albumArtURL: String?
    public var positionSeconds: Double?
    public var durationSeconds: Double?
    public var volume: Int?
    public var muted: Bool?
    public var hudEnabled: Bool?
    public var command: String?
    public var message: String?

    public init(
        type: String,
        protocol: Int? = nil,
        room: String? = nil,
        state: String? = nil,
        title: String? = nil,
        artist: String? = nil,
        album: String? = nil,
        albumArtURL: String? = nil,
        positionSeconds: Double? = nil,
        durationSeconds: Double? = nil,
        volume: Int? = nil,
        muted: Bool? = nil,
        hudEnabled: Bool? = nil,
        command: String? = nil,
        message: String? = nil
    ) {
        self.type = type
        self.protocol = `protocol`
        self.room = room
        self.state = state
        self.title = title
        self.artist = artist
        self.album = album
        self.albumArtURL = albumArtURL
        self.positionSeconds = positionSeconds
        self.durationSeconds = durationSeconds
        self.volume = volume
        self.muted = muted
        self.hudEnabled = hudEnabled
        self.command = command
        self.message = message
    }

    public static func ready() -> IPCMessage {
        IPCMessage(type: "ready")
    }

    public static func command(_ command: String) -> IPCMessage {
        IPCMessage(type: "command", command: command)
    }

    public static func error(_ message: String) -> IPCMessage {
        IPCMessage(type: "error", message: message)
    }
}

public func encodeLine(_ message: IPCMessage) throws -> Data {
    let encoded = try JSONEncoder().encode(message)
    var out = encoded
    out.append(0x0a)
    return out
}

public func decodeLine(_ data: Data) throws -> IPCMessage {
    var trimmed = data
    while trimmed.last == 0x0a || trimmed.last == 0x0d {
        trimmed.removeLast()
    }
    return try JSONDecoder().decode(IPCMessage.self, from: trimmed)
}
