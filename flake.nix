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
            hash = "sha256-s5ST8VDMv9nO//7xYY8ejIw4+dm5i6IrLCr0i2FrM00=";
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
            hash = "sha256-s5ST8VDMv9nO//7xYY8ejIw4+dm5i6IrLCr0i2FrM00=";
            fetcherVersion = 3;
          };
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
