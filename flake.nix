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
        lib = nixpkgs.legacyPackages.${system}.lib;
        
        # Go toolchain (latest stable, compatible with go 1.23+)
        go = pkgs.go;
        
        # mozjpeg is available in nixpkgs
        mozjpeg = pkgs.mozjpeg;
        
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
          
          # Shell hook for additional setup
          shellHook = ''
            # mozjpeg/libpng/zlib への RUNPATH をビルド成果物に埋め込む。
            # LD_LIBRARY_PATH を設定しないことで、この devShell 内で起動する
            # 他のツール（serena MCP など）のライブラリ解決を汚染しない。
            # mkShell が buildInputs から自動生成する LD_LIBRARY_PATH を解除する。
            unset LD_LIBRARY_PATH
            export NIX_LDFLAGS="-rpath ${mozjpeg}/lib -rpath ${pkgs.libpng}/lib -rpath ${pkgs.zlib}/lib ''${NIX_LDFLAGS:-}"
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
          
          vendorHash = "sha256-fgcugDy2vbDz5nqjjB/my8c+Oa4/Rs+gsgHASuiuOIQ=";
          
          buildInputs = [ mozjpeg pkgs.libpng pkgs.zlib ];
          
          # CGO flags for libjpeg_turbo
          env = {
            CGO_ENABLED = "1";
            CGO_CFLAGS = "-I${mozjpeg}/include";
            CGO_LDFLAGS = "-L${mozjpeg}/lib -ljpeg";
          };
        };
      }
    );
}