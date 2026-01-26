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
        in
        {
          clankers-daemon = pkgs.buildGoModule {
            pname = "clankers-daemon";
            version = "0.1.0";
            src = ./packages/daemon;
            vendorHash = "sha256-L8CHwPOjwE+DOJ1OWi0/V+tYrB2ev3iN9VU7i8WmCN0=";

            env = {
              CGO_ENABLED = 0;
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
