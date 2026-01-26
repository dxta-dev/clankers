{
  description = "clankers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Cross-compilation targets for daemon releases
      daemonTargets = [
        {
          name = "linux-amd64";
          GOOS = "linux";
          GOARCH = "amd64";
          suffix = "";
        }
        {
          name = "linux-arm64";
          GOOS = "linux";
          GOARCH = "arm64";
          suffix = "";
        }
        {
          name = "darwin-amd64";
          GOOS = "darwin";
          GOARCH = "amd64";
          suffix = "";
        }
        {
          name = "darwin-arm64";
          GOOS = "darwin";
          GOARCH = "arm64";
          suffix = "";
        }
        {
          name = "windows-amd64";
          GOOS = "windows";
          GOARCH = "amd64";
          suffix = ".exe";
        }
      ];
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };

          # Shared pnpm dependencies for all TypeScript packages
          pnpmDeps = pkgs.fetchPnpmDeps {
            pname = "clankers-workspace";
            version = "0.1.0";
            src = ./.;
            hash = "sha256-szJy9JkSlOYT7aCa3mfrXajbHDWpTZcQkzQdj7eiW8Q=";
            fetcherVersion = 3;
          };

          # Helper to build TypeScript apps
          mkTsApp =
            {
              pname,
              appDir,
              filterName,
            }:
            pkgs.stdenv.mkDerivation {
              inherit pname;
              version = "0.1.0";
              src = ./.;

              nativeBuildInputs = [
                pkgs.nodejs_24
                pkgs.pnpm
                pkgs.pnpmConfigHook
              ];

              inherit pnpmDeps;

              buildPhase = ''
                runHook preBuild
                pnpm --filter ${filterName} build
                runHook postBuild
              '';

              installPhase = ''
                runHook preInstall
                mkdir -p $out
                cp -r ${appDir}/dist $out/
                cp -r ${appDir}/src $out/
                cp ${appDir}/package.json $out/
                runHook postInstall
              '';

              meta = {
                description = "Clankers plugin for ${pname}";
              };
            };

          # Helper to build daemon for a specific target (cross-compilation)
          # buildGoModule uses GOOS/GOARCH from env, but we need to set them
          # via preBuild to ensure they're used during the build phase
          mkDaemonCross =
            {
              name,
              GOOS,
              GOARCH,
              suffix,
            }:
            pkgs.buildGoModule {
              pname = "clankers-daemon-${name}";
              version = "0.1.0";
              src = ./packages/daemon;
              vendorHash = "sha256-L8CHwPOjwE+DOJ1OWi0/V+tYrB2ev3iN9VU7i8WmCN0=";

              # Strip debug symbols and DWARF info to reduce binary size
              ldflags = [
                "-s"
                "-w"
              ];

              # Remove file paths from binary for reproducibility
              flags = [ "-trimpath" ];

              # Disable fixup phases that don't work on cross-compiled binaries
              dontStrip = true;
              dontPatchELF = true;
              dontFixup = true;

              # Set cross-compilation environment
              preBuild = ''
                export GOOS=${GOOS}
                export GOARCH=${GOARCH}
                export CGO_ENABLED=0
              '';

              # Move binary from GOOS_GOARCH subdir to bin/ and handle Windows .exe
              postInstall =
                let
                  # Windows builds produce .exe in the subdir
                  srcBinary = if GOOS == "windows" then "clankers-daemon.exe" else "clankers-daemon";
                  dstBinary = "clankers-daemon${suffix}";
                in
                ''
                  if [ -d "$out/bin/${GOOS}_${GOARCH}" ]; then
                    mv "$out/bin/${GOOS}_${GOARCH}/${srcBinary}" "$out/bin/${dstBinary}"
                    rmdir "$out/bin/${GOOS}_${GOARCH}"
                  elif [ -f "$out/bin/clankers-daemon" ] && [ "${suffix}" != "" ]; then
                    mv "$out/bin/clankers-daemon" "$out/bin/${dstBinary}"
                  fi
                '';

              meta = {
                description = "Clankers daemon for ${name}";
                mainProgram = "clankers-daemon${suffix}";
              };
            };

          # Generate cross-compiled daemon packages
          daemonCrossPackages = builtins.listToAttrs (
            map (target: {
              name = "clankers-daemon-${target.name}";
              value = mkDaemonCross target;
            }) daemonTargets
          );
        in
        daemonCrossPackages
        // {
          clankers-daemon = pkgs.buildGoModule {
            pname = "clankers-daemon";
            version = "0.1.0";
            src = ./packages/daemon;
            vendorHash = "sha256-L8CHwPOjwE+DOJ1OWi0/V+tYrB2ev3iN9VU7i8WmCN0=";

            # Strip debug symbols and DWARF info to reduce binary size
            ldflags = [
              "-s"
              "-w"
            ];

            # Remove file paths from binary for reproducibility
            flags = [ "-trimpath" ];

            env = {
              CGO_ENABLED = "0";
            };

            meta = {
              description = "Clankers daemon - SQLite persistence for AI harness plugins";
              mainProgram = "clankers-daemon";
            };
          };

          clankers-opencode = mkTsApp {
            pname = "clankers-opencode";
            appDir = "apps/opencode-plugin";
            filterName = "@dxta-dev/clankers-opencode";
          };

          clankers-cursor = mkTsApp {
            pname = "clankers-cursor";
            appDir = "apps/cursor-plugin";
            filterName = "@dxta-dev/clankers-cursor";
          };

          clankers-claude-code = mkTsApp {
            pname = "clankers-claude-code";
            appDir = "apps/claude-code-plugin";
            filterName = "@dxta-dev/clankers-claude-code";
          };

          default = self.packages.${system}.clankers-daemon;
        }
      );

      checks = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };

          # Shared pnpm dependencies for typecheck
          pnpmDeps = pkgs.fetchPnpmDeps {
            pname = "clankers-workspace";
            version = "0.1.0";
            src = ./.;
            hash = "sha256-szJy9JkSlOYT7aCa3mfrXajbHDWpTZcQkzQdj7eiW8Q=";
            fetcherVersion = 3;
          };

          daemon = self.packages.${system}.clankers-daemon;
        in
        {
          lint = pkgs.stdenvNoCC.mkDerivation {
            name = "clankers-lint";
            src = ./.;

            nativeBuildInputs = [ pkgs.biome ];

            buildPhase = ''
              biome lint .
            '';

            installPhase = ''
              touch $out
            '';
          };

          typecheck = pkgs.stdenvNoCC.mkDerivation {
            name = "clankers-typecheck";
            src = ./.;

            nativeBuildInputs = [
              pkgs.nodejs_24
              pkgs.pnpm
              pkgs.pnpmConfigHook
            ];

            inherit pnpmDeps;

            buildPhase = ''
              pnpm check
            '';

            installPhase = ''
              touch $out
            '';
          };

          integration = pkgs.stdenvNoCC.mkDerivation {
            name = "clankers-integration";
            src = ./.;

            nativeBuildInputs = [
              pkgs.nodejs_24
              pkgs.pnpm
              pkgs.pnpmConfigHook
              daemon
            ];

            inherit pnpmDeps;

            buildPhase = ''
              # Create isolated test directory
              TEST_DIR=$(mktemp -d)
              export CLANKERS_SOCKET_PATH="$TEST_DIR/clankers.sock"
              export CLANKERS_DB_PATH="$TEST_DIR/clankers.db"

              cleanup() {
                echo "Cleaning up..."
                if [ -n "''${DAEMON_PID:-}" ]; then
                  kill "$DAEMON_PID" 2>/dev/null || true
                  wait "$DAEMON_PID" 2>/dev/null || true
                fi
                rm -rf "$TEST_DIR"
              }
              trap cleanup EXIT

              echo "Starting daemon..."
              echo "  Socket: $CLANKERS_SOCKET_PATH"
              echo "  DB: $CLANKERS_DB_PATH"

              clankers-daemon &
              DAEMON_PID=$!

              # Wait for socket to be ready
              for i in $(seq 1 30); do
                if [ -S "$CLANKERS_SOCKET_PATH" ]; then
                  echo "Daemon ready after $i attempts"
                  break
                fi
                sleep 0.1
              done

              if [ ! -S "$CLANKERS_SOCKET_PATH" ]; then
                echo "ERROR: Daemon failed to start (socket not found)"
                exit 1
              fi

              echo ""
              echo "Running integration tests..."
              pnpm exec tsx tests/integration.ts
            '';

            installPhase = ''
              touch $out
            '';
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.nodejs_24
              pkgs.pnpm
              pkgs.nodePackages.typescript
              pkgs.nodePackages.typescript-language-server

              pkgs.go

              pkgs.sqlite

              pkgs.biome
            ];

            shellHook = ''
              echo "Clankers dev shell loaded"
              echo "  Node: $(node --version)"
              echo "  pnpm: $(pnpm --version)"
              echo "  Go:   $(go version | cut -d' ' -f3)"
            '';
          };
        }
      );
    };
}
