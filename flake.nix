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

      clankersOverlay = final: prev: {
        clankers-daemon = final.callPackage (
          { buildGoModule }:
          buildGoModule {
            pname = "clankers-daemon";
            version = "0.1.0";
            src = ./packages/daemon;
            vendorHash = "sha256-L8CHwPOjwE+DOJ1OWi0/V+tYrB2ev3iN9VU7i8WmCN0=";

            ldflags = [
              "-s"
              "-w"
            ];

            flags = [ "-trimpath" ];

            env = {
              CGO_ENABLED = "0";
            };

            meta = {
              description = "Clankers daemon - SQLite persistence for AI harness plugins";
              mainProgram = "clankers-daemon";
              homepage = "https://github.com/dxta-dev/clankers";
              license = final.lib.licenses.mit;
            };
          }
        ) { };
      };

      nixosModule =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.services.clankers-daemon;
        in
        {
          options.services.clankers-daemon = {
            enable = lib.mkEnableOption "clankers-daemon - SQLite persistence service for AI harness plugins";

            package = lib.mkOption {
              type = lib.types.package;
              default = pkgs.clankers-daemon or self.packages.${pkgs.system}.clankers-daemon;
              defaultText = lib.literalExpression "pkgs.clankers-daemon";
              description = "The clankers-daemon package to use";
            };

            dataRoot = lib.mkOption {
              type = lib.types.str;
              default = "%S/clankers";
              description = ''
                Data root directory for the daemon.
                %S expands to /var/lib for system services.
                The database and socket will be created here.
              '';
            };

            dbPath = lib.mkOption {
              type = lib.types.nullOr lib.types.str;
              default = null;
              description = ''
                Explicit database file path. If not set, uses {dataRoot}/clankers.db.
              '';
            };

            socketPath = lib.mkOption {
              type = lib.types.nullOr lib.types.str;
              default = null;
              description = ''
                Explicit socket path. If not set, uses {dataRoot}/dxta-clankers.sock.
              '';
            };

            logLevel = lib.mkOption {
              type = lib.types.enum [
                "debug"
                "info"
                "warn"
                "error"
              ];
              default = "info";
              description = "Log level for the daemon";
            };

            user = lib.mkOption {
              type = lib.types.str;
              default = "clankers";
              description = "User to run the daemon as";
            };

            group = lib.mkOption {
              type = lib.types.str;
              default = "clankers";
              description = "Group to run the daemon as";
            };
          };

          config = lib.mkIf cfg.enable {
            users.users.${cfg.user} = {
              isSystemUser = true;
              group = cfg.group;
              home = "/var/lib/clankers";
              createHome = true;
            };

            users.groups.${cfg.group} = { };

            systemd.services.clankers-daemon = {
              description = "Clankers Daemon - SQLite persistence for AI harness plugins";
              after = [ "network.target" ];
              wantedBy = [ "multi-user.target" ];

              serviceConfig = {
                Type = "simple";
                User = cfg.user;
                Group = cfg.group;
                ExecStart =
                  let
                    args = lib.concatStringsSep " " [
                      "--log-level=${cfg.logLevel}"
                      (lib.optionalString (cfg.dataRoot != "") "--data-root=${cfg.dataRoot}")
                      (lib.optionalString (cfg.dbPath != null) "--db-path=${cfg.dbPath}")
                      (lib.optionalString (cfg.socketPath != null) "--socket=${cfg.socketPath}")
                    ];
                  in
                  "${cfg.package}/bin/clankers-daemon ${args}";

                Restart = "on-failure";
                RestartSec = 5;

                NoNewPrivileges = true;
                PrivateTmp = true;
                ProtectSystem = "strict";
                ProtectHome = true;
                ReadWritePaths = [ cfg.dataRoot ];
              };

              environment = {
                CLANKERS_DATA_PATH = cfg.dataRoot;
              }
              // lib.optionalAttrs (cfg.dbPath != null) {
                CLANKERS_DB_PATH = cfg.dbPath;
              }
              // lib.optionalAttrs (cfg.socketPath != null) {
                CLANKERS_SOCKET_PATH = cfg.socketPath;
              };
            };
          };
        };

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

          pnpmDeps = pkgs.fetchPnpmDeps {
            pname = "clankers-workspace";
            version = "0.1.0";
            src = ./.;
            hash = "sha256-szJy9JkSlOYT7aCa3mfrXajbHDWpTZcQkzQdj7eiW8Q=";
            fetcherVersion = 3;
          };

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

              ldflags = [
                "-s"
                "-w"
              ];

              flags = [ "-trimpath" ];

              dontStrip = true;
              dontPatchELF = true;
              dontFixup = true;

              preBuild = ''
                export GOOS=${GOOS}
                export GOARCH=${GOARCH}
                export CGO_ENABLED=0
              '';

              postInstall =
                let
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

            ldflags = [
              "-s"
              "-w"
            ];

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
          daemon = self.packages.${system}.clankers-daemon;
          opencodePlugin = self.packages.${system}.clankers-opencode;
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

              daemon
            ];

            shellHook = ''
              export CLANKERS_DATA_PATH="$PWD/.clankers-dev"
              export CLANKERS_SOCKET_PATH="$PWD/.clankers-dev/dxta-clankers.sock"
              export CLANKERS_DB_PATH="$PWD/.clankers-dev/clankers.db"

              echo "Clankers dev shell loaded"
              echo "  Node: $(node --version)"
              echo "  pnpm: $(pnpm --version)"
              echo "  Go:   $(go version | cut -d' ' -f3)"
              echo ""
              echo "Dev environment paths:"
              echo "  Data: $CLANKERS_DATA_PATH"
              echo "  Socket: $CLANKERS_SOCKET_PATH"
              echo "  DB: $CLANKERS_DB_PATH"
              echo ""
              echo "Commands:"
              echo "  clankers-daemon              - Start daemon manually"
              echo "  clankers-daemon --help       - Show daemon options"
              echo ""
              echo "Other dev shells:"
              echo "  nix develop .#with-daemon-and-plugin - Dev shell with auto-started daemon + plugin setup"
              echo ""

              # Create dev data directory
              mkdir -p "$CLANKERS_DATA_PATH"

              # Check if daemon is already running
              if [ -S "$CLANKERS_SOCKET_PATH" ]; then
                echo "Daemon appears to be running (socket exists)"
              else
                echo "Daemon not running. Start with: clankers-daemon &"
              fi
            '';
          };

          with-plugin = pkgs.mkShell {
            packages = [
              pkgs.nodejs_24
              pkgs.pnpm
              pkgs.nodePackages.typescript
              pkgs.nodePackages.typescript-language-server
              pkgs.go
              pkgs.sqlite
              pkgs.biome
              daemon
            ];

            shellHook = ''
              export CLANKERS_DATA_PATH="$PWD/.clankers-dev"
              export CLANKERS_SOCKET_PATH="$PWD/.clankers-dev/dxta-clankers.sock"
              export CLANKERS_DB_PATH="$PWD/.clankers-dev/clankers.db"

              echo "Clankers dev shell (with plugin setup) loaded"
              echo "  Node: $(node --version)"
              echo "  pnpm: $(pnpm --version)"
              echo "  Go:   $(go version | cut -d' ' -f3)"
              echo ""

              # Create dev data directory
              mkdir -p "$CLANKERS_DATA_PATH"

              # Create .opencode directory structure
              mkdir -p "$PWD/.opencode/plugins"

              # Build and copy the plugin
              echo "Building OpenCode plugin..."
              if pnpm --filter @dxta-dev/clankers-opencode build 2>/dev/null; then
                if [ -f "$PWD/apps/opencode-plugin/dist/index.js" ]; then
                  cp "$PWD/apps/opencode-plugin/dist/index.js" "$PWD/.opencode/plugins/clankers.js"
                  echo "Plugin built and copied to .opencode/plugins/clankers.js"
                else
                  echo "Warning: Build succeeded but output not found"
                fi
              else
                echo "Warning: Plugin build failed (may need pnpm install)"
              fi

              # Create project-level config if it doesn't exist
              if [ ! -f "$PWD/.opencode/config.json" ]; then
                echo '{"$schema":"https://opencode.ai/config.json","plugin":["./plugins/clankers.js"]}' > "$PWD/.opencode/config.json"
                echo "Created .opencode/config.json"
              fi

              # Check if daemon is already running
              if [ -S "$CLANKERS_SOCKET_PATH" ]; then
                echo ""
                echo "Daemon appears to be running (socket exists)"
              else
                echo ""
                echo "Daemon not running. Start with: clankers-daemon &"
              fi

              echo ""
              echo "Setup complete! Project-level plugin is ready."
              echo "  Config: $PWD/.opencode/config.json"
              echo "  Plugin: $PWD/.opencode/plugins/clankers.js"
              echo ""
              echo "Restart OpenCode in this directory to load the local plugin."
            '';
          };

          with-daemon-and-plugin = pkgs.mkShell {
            packages = [
              pkgs.nodejs_24
              pkgs.pnpm
              pkgs.nodePackages.typescript
              pkgs.nodePackages.typescript-language-server
              pkgs.go
              pkgs.sqlite
              pkgs.biome
              daemon
            ];

            shellHook = ''
              export CLANKERS_DATA_PATH="$PWD/.clankers-dev"
              export CLANKERS_SOCKET_PATH="$PWD/.clankers-dev/dxta-clankers.sock"
              export CLANKERS_DB_PATH="$PWD/.clankers-dev/clankers.db"

              echo "Clankers dev shell (with daemon + plugin) loaded"
              echo "  Node: $(node --version)"
              echo "  pnpm: $(pnpm --version)"
              echo "  Go:   $(go version | cut -d' ' -f3)"
              echo ""

              # Create dev data directory
              mkdir -p "$CLANKERS_DATA_PATH"

              # Kill any existing daemon on this socket
              if [ -S "$CLANKERS_SOCKET_PATH" ]; then
                echo "Cleaning up old socket..."
                rm -f "$CLANKERS_SOCKET_PATH"
              fi

              # Start daemon in background
              echo "Starting clankers-daemon..."
              clankers-daemon --log-level=debug &
              DAEMON_PID=$!

              # Store PID for cleanup
              echo $DAEMON_PID > "$CLANKERS_DATA_PATH/daemon.pid"

              # Wait for socket
              for i in $(seq 1 30); do
                if [ -S "$CLANKERS_SOCKET_PATH" ]; then
                  echo "Daemon ready (PID: $DAEMON_PID)"
                  break
                fi
                sleep 0.1
              done

              if [ ! -S "$CLANKERS_SOCKET_PATH" ]; then
                echo "WARNING: Daemon may not have started properly"
              fi

              # Create .opencode directory structure
              mkdir -p "$PWD/.opencode/plugins"

              # Build and copy the plugin
              echo ""
              echo "Building OpenCode plugin..."
              if pnpm --filter @dxta-dev/clankers-opencode build 2>/dev/null; then
                if [ -f "$PWD/apps/opencode-plugin/dist/index.js" ]; then
                  cp "$PWD/apps/opencode-plugin/dist/index.js" "$PWD/.opencode/plugins/clankers.js"
                  echo "Plugin built and copied to .opencode/plugins/clankers.js"
                else
                  echo "Warning: Build succeeded but output not found"
                fi
              else
                echo "Warning: Plugin build failed (may need pnpm install)"
              fi

              # Create project-level config if it doesn't exist
              if [ ! -f "$PWD/.opencode/config.json" ]; then
                echo '{"$schema":"https://opencode.ai/config.json","plugin":["./plugins/clankers.js"]}' > "$PWD/.opencode/config.json"
                echo "Created .opencode/config.json"
              fi

              echo ""
              echo "Socket: $CLANKERS_SOCKET_PATH"
              echo "Config: $PWD/.opencode/config.json"
              echo "Plugin: $PWD/.opencode/plugins/clankers.js"
              echo ""
              echo "The daemon will stop when you exit this shell."
              echo "Restart OpenCode in this directory to load the local plugin."
              echo ""

              # Set up cleanup on shell exit
              cleanup_daemon() {
                if [ -f "$CLANKERS_DATA_PATH/daemon.pid" ]; then
                  local pid=$(cat "$CLANKERS_DATA_PATH/daemon.pid")
                  if kill -0 "$pid" 2>/dev/null; then
                    echo ""
                    echo "Stopping daemon (PID: $pid)..."
                    kill "$pid" 2>/dev/null || true
                    wait "$pid" 2>/dev/null || true
                  fi
                  rm -f "$CLANKERS_DATA_PATH/daemon.pid"
                fi
              }
              trap cleanup_daemon EXIT
            '';
          };
        }
      );

      overlays.default = clankersOverlay;

      nixosModules.default = nixosModule;
    };
}
