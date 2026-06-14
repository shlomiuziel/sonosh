// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "sonosh-helper",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "sonosh-macos-helper", targets: ["sonosh-macos-helper"])
    ],
    targets: [
        .target(name: "SonoshHelperCore"),
        .executableTarget(
            name: "sonosh-macos-helper",
            dependencies: ["SonoshHelperCore"]
        )
    ]
)
