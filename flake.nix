{
  description = "claudecode-hooks";

  inputs = {
    nixpkgs.url = "nixpkgs/0f8bab2331d06f7b685f16737a057f68eb448ab6";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go_1_24;
      in
      {
        devShell = pkgs.mkShellNoCC {
          name = "claudecode-hooks";
          nativeBuildInputs = with pkgs; [
            go

            # For testing the bash block hook
            awscli2
            kubectl
          ];
          CGO_ENABLED = 0;
        };
      }
    );
}
