{
  description = "slides (fork adaptado: LaTeX, imágenes e imágenes con pie de foto, bibliografía)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools # staticcheck
            delve

            # LaTeX -> PNG pipeline
            tectonic
            ghostscript
            imagemagick

            # ejecución de código en las slides: Ctrl+E
            julia-bin

            # comodidad
            git
          ];

          shellHook = ''
            export GOFLAGS="-mod=mod"
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "slides";
          version = "dev";
          src = self;

          vendorHash = "sha256-NTO7H0aLudHn83oARrmsEfhz+4cFhHbEXjQno5JmzGM=";

          # Los tests que ejecutan procesos externos (tectonic, gs) se
          # excluyen del build sandboxeado de Nix.
          checkFlags = [ "-short" ];

          meta = {
            description = "Terminal slides, fork personal con soporte de LaTeX/imágenes/bibliografía";
            mainProgram = "slides";
          };
        };
      }
    );
}
