{
  description = "Ebiten flake template";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go
            libGL
            xorg.libX11
            xorg.libXrandr
            xorg.libXcursor
            xorg.libXinerama
            xorg.libXi
            xorg.libXxf86vm
          ];

          name = "ebitengine";

          shellHook = ''
            export LD_LIBRARY_PATH=${pkgs.wayland}/lib:${pkgs.lib.getLib pkgs.libGL}/lib:${pkgs.lib.getLib pkgs.libGL}/lib:$LD_LIBRARY_PATH
          '';
        };
      }
    );
}
