{
  description = "Cyber range control plane";

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
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "controlplane";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
        };

        checks = {
          build = self.packages.${system}.default;

          # tests + vet as a derivation
          tests = pkgs.buildGoModule {
            pname = "controlplane-tests";
            version = "0.1.0";
            src = ./.;
            vendorHash = null;
            checkPhase = ''
              go test ./...
              go vet ./...
            '';
          };

          lint =
            pkgs.runCommand "golangci-lint"
              {
                buildInputs = [
                  pkgs.go
                  pkgs.golangci-lint
                ];
              }
              ''
                cp -r ${./.} src && cd src
                 export HOME=$TMPDIR
                 export CGO_ENABLED=0
                 export GOFLAGS=-mod=mod
                 golangci-lint run ./... && touch $out
              '';
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            golangci-lint
            opentofu
          ];
        };
      }
    );
}
