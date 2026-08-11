{
  description = "Jaipur - Image archive compression tool with mozjpeg";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Go toolchain (latest stable, compatible with go 1.23+)
        go = pkgs.go;
        
        # mozjpeg is not in nixpkgs; use libjpeg_turbo (compatible)
        mozjpeg = pkgs.libjpeg_turbo;
        
        # Development tools
        devTools = with pkgs; [
          # Go toolchain
          go
          gopls
          go-tools
          delve
          
          # Build tools
          pkg-config
          gcc
          gnumake
          
          # Image processing libraries
          mozjpeg
          libpng
          zlib
          
          # Archive libraries
          libarchive
          bzip2
          xz
          zstd
          
          # Development utilities
          git
          jq
          curl
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          name = "jaipur-dev";
          
          buildInputs = devTools;
          
          # Environment variables
          CGO_ENABLED = "1";
          
          # pkg-config paths for libjpeg_turbo
          PKG_CONFIG_PATH = "${mozjpeg}/lib/pkgconfig:${pkgs.libpng}/lib/pkgconfig:${pkgs.zlib}/lib/pkgconfig";
          
          # Library paths for runtime
          LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath [
            mozjpeg
            pkgs.libpng
            pkgs.zlib
            pkgs.libarchive
            pkgs.bzip2
            pkgs.xz
            pkgs.zstd
          ];
          
          # Shell hook for additional setup
          shellHook = ''
            echo "🦀 Jaipur development environment"
            echo "Go version: $(go version)"
            echo "libjpeg_turbo: ${mozjpeg}"
            echo ""
            echo "Available commands:"
            echo "  go build    - Build the project"
            echo "  go test     - Run tests"
            echo "  go run .    - Run the application"
            echo ""
          '';
        };
        
        # Optional: package definition
        packages.default = pkgs.buildGoModule {
          pname = "jaipur";
          version = "0.1.0";
          src = ./.;
          
          vendorHash = null; # or specify hash
          
          buildInputs = [ mozjpeg pkgs.libpng pkgs.zlib ];
          
          # CGO flags for libjpeg_turbo
          CGO_ENABLED = "1";
          CGO_CFLAGS = "-I${mozjpeg}/include";
          CGO_LDFLAGS = "-L${mozjpeg}/lib -ljpeg";
        };
      }
    );
}