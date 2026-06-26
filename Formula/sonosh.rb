class Sonosh < Formula
  desc "Sonos TUI and CLI"
  homepage "https://github.com/shlomiuziel/sonosh"
  head "https://github.com/shlomiuziel/sonosh.git", branch: "main"

  depends_on "go" => :build

  def install
    sonosh_bin = libexec/"sonosh"
    system "go", "build", "-o", sonosh_bin, "./cmd/sonosh"

    if OS.mac?
      helper_path = buildpath/"helpers/macos/sonosh-helper"
      system "swift", "build", "--package-path", helper_path, "--configuration", "release"
      libexec.install helper_path/".build/release/sonosh-macos-helper"
    end

    (bin/"sonosh").write <<~EOS
      #!/bin/bash
      HELPER="#{libexec}/sonosh-macos-helper"
      if [ -x "$HELPER" ]; then
        export SONOSH_MAC_HELPER="$HELPER"
      fi
      exec "#{sonosh_bin}" "$@"
    EOS
    chmod 0755, bin/"sonosh"
  end
end
