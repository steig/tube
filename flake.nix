{
  description = "tube - Local development proxy with .test domains and Cloudflare tunnels";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go
            go
            gotools
            golangci-lint

            # System dependencies for proxy
            nginx
            dnsmasq
            cloudflared
            mkcert

            # Development tools
            git
            just
            air # Hot reload for Go
            goreleaser

            # Optional: for building and debugging
            delve # Go debugger
            postgresql # For integration tests if needed
          ];

          shellHook = ''
            # Set up environment variables
            export GOFLAGS="-mod=mod"

            # Create bin directory if it doesn't exist
            mkdir -p "$PWD/bin"

            # Add local bin to PATH
            export PATH="$PWD/bin:$PATH"

            # Print welcome message
            echo "🚀 tube development environment loaded"
            echo ""
            echo "Quick start:"
            echo "  just dev                 # Start with hot reload"
            echo "  just build               # Build the binary"
            echo "  just test                # Run tests"
            echo ""
            echo "All available commands:"
            just --list
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "tube";
          version = "0.1.0-dev";
          src = ./.;
          vendorHash = null; # Can be set to a hash after first build

          subPackages = [ "cmd/tube" ];

          ldflags = [
            "-s"
            "-w"
            "-X main.Version=${self.shortRev or "dev"}"
            "-X main.Commit=${self.shortRev or "unknown"}"
            "-X main.Date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
          ];

          meta = with pkgs.lib; {
            description = "Local development proxy with .test domains and Cloudflare tunnels";
            homepage = "https://github.com/steig/tube";
            license = licenses.mit;
            maintainers = [ ];
          };
        };
      }
    );
}
