{ nixpkgs ? import <nixpkgs> {}}:
let
  inherit (nixpkgs) pkgs;
  python = pkgs.python36.withPackages (ps: with ps; [
    jupyter
    scikitlearn
    pandas
    numpy
    matplotlib
  ]);
in
  pkgs.stdenv.mkDerivation {
    name = "jupyter-env";
    buildInputs = [ python pkgs.python36Packages.jupyter ];
  }
