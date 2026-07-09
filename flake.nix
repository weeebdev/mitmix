{
  description = "mitm-decentralized dev shell (Go hub + Python mitmproxy agents)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go_1_25
            pkgs.python311
            pkgs.python311Packages.pip
            pkgs.python311Packages.virtualenv
            pkgs.docker-compose
            pkgs.mitmproxy
          ];
          shellHook = ''
            echo "mitm-decentralized dev shell"
            echo "hub: go run .   |   agent: cd agent && pip install -e ."
          '';
        };
      }
    );
}
