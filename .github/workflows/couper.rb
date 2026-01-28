class Couper < Formula
    desc "Couper is a lightweight open source API gateway designed to support developers in building and running API-driven Web projects."
    homepage "https://couper.io/"
    license "MIT"
    version "{{ GITHUB_REF_NAME }}"
    head "https://github.com/coupergateway/couper.git", branch: "main"

    on_macos do
      if Hardware::CPU.arm?
        url "https://github.com/coupergateway/couper/releases/download/{{ GITHUB_REF_NAME }}/couper-{{ GITHUB_REF_NAME }}-macos-arm64.zip"
        sha256 "{{ MACOS_ARM64_SHA256 }}"

        def install
          bin.install "couper"
        end
      end
      if Hardware::CPU.intel?
        url "https://github.com/coupergateway/couper/releases/download/{{ GITHUB_REF_NAME }}/couper-{{ GITHUB_REF_NAME }}-macos-amd64.zip"
        sha256 "{{ MACOS_AMD64_SHA256 }}"

        def install
          bin.install "couper"
        end
      end
    end

    on_linux do
      if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
        url "https://github.com/coupergateway/couper/releases/download/{{ GITHUB_REF_NAME }}/couper-{{ GITHUB_REF_NAME }}-linux-arm64.tar.gz"
        sha256 "{{ LINUX_ARM64_SHA256 }}"

        def install
          bin.install "couper"
        end
      end
      if Hardware::CPU.intel?
        url "https://github.com/coupergateway/couper/releases/download/{{ GITHUB_REF_NAME }}/couper-{{ GITHUB_REF_NAME }}-linux-amd64.tar.gz"
        sha256 "{{ LINUX_AMD64_SHA256 }}"

        def install
          bin.install "couper"
        end
      end
    end

    test do
      system "#{bin}/couper version"
    end
  end
