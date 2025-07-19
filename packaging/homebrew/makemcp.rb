class Makemcp < Formula
  desc "CLI tool that creates MCP servers from various sources"
  homepage "https://github.com/T4cceptor/MakeMCP"
  url "https://github.com/T4cceptor/MakeMCP/archive/v#{version}.tar.gz"
  license "Apache-2.0"
  head "https://github.com/T4cceptor/MakeMCP.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/makemcp.go"
  end

  test do
    assert_match "MakeMCP", shell_output("#{bin}/makemcp --help")
    
    # Test version command
    assert_match version.to_s, shell_output("#{bin}/makemcp --version")
  end
end