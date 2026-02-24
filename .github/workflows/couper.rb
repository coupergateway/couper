class Couper < Formula
    desc "Couper is a lightweight open source API gateway designed to support developers in building and running API-driven Web projects."
    homepage "https://couper.io/"
    license "MIT"
    version "{{ GITHUB_REF_NAME }}"
    url "https://github.com/coupergateway/couper/archive/refs/tags/{{ GITHUB_REF_NAME }}.tar.gz"
    sha256 "{{ SOURCE_SHA256 }}"
    head "https://github.com/coupergateway/couper.git", branch: "main"

    depends_on "go" => :build

    def install
      ldflags = %W[
        -X github.com/coupergateway/couper/utils.VersionName=#{version}
        -X github.com/coupergateway/couper/utils.BuildName={{ SHORT_SHA }}
        -X github.com/coupergateway/couper/utils.BuildDate=#{time.strftime("%F")}
      ]
      system "go", "build", *std_go_args(ldflags:)
    end

    test do
      system bin/"couper", "version"
    end
  end
