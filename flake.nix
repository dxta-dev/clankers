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
