# Homebrew Formula for scribbles
# To use this formula:
#   1. Create a tap: brew tap jfmyers9/scribbles
#   2. Copy this file to the tap repository
#   3. Update the URL and SHA256 for each release
#
# Users can then install with:
#   brew install jfmyers9/scribbles/scribbles

class Scribbles < Formula
  desc "Apple Music scrobbler for Last.fm"
  homepage "https://github.com/jfmyers9/scribbles"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jfmyers9/scribbles/releases/download/v#{version}/scribbles-v#{version}-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ARM64_SHA256"
    else
      url "https://github.com/jfmyers9/scribbles/releases/download/v#{version}/scribbles-v#{version}-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_AMD64_SHA256"
    end
  end

  def install
    # Determine the correct binary based on architecture
    if Hardware::CPU.arm?
      bin.install "scribbles-darwin-arm64" => "scribbles"
    else
      bin.install "scribbles-darwin-amd64" => "scribbles"
    end
  end

  def caveats
    <<~EOS
      To get started with scribbles:

      1. Authenticate with Last.fm:
           scribbles auth

      2. Install the background daemon:
           scribbles install

      3. The daemon will automatically start and scrobble your Apple Music playback.

      Configuration is stored in: ~/.config/scribbles/config.yaml
      Logs are stored in: ~/.local/share/scribbles/logs/

      To use in tmux status line, add to your .tmux.conf:
           set -g status-right "#(scribbles now)"

      For more information, visit:
           https://github.com/jfmyers9/scribbles
    EOS
  end

  test do
    system "#{bin}/scribbles", "--version"
  end
end
