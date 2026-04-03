# frozen_string_literal: true

class Celador < Formula
  desc "Zero-trust dependency security CLI for JS/TS workspaces"
  homepage "https://github.com/GustavoGutierrez/celador"
  license "MIT"
  version "__VERSION__"

  on_macos do
    depends_on arch: :arm64

    url "https://github.com/GustavoGutierrez/celador/releases/download/v#{version}/celador_#{version}_darwin_arm64.tar.gz"
    sha256 "__DARWIN_ARM64_SHA256__"
  end

  on_linux do
    depends_on arch: :x86_64

    url "https://github.com/GustavoGutierrez/celador/releases/download/v#{version}/celador_#{version}_linux_amd64.tar.gz"
    sha256 "__LINUX_AMD64_SHA256__"
  end

  def install
    bin.install "celador"
  end

  test do
    assert_match "Zero-trust dependency security", shell_output("#{bin}/celador --help")
  end
end
