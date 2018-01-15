{ nixpkgs ? import <nixpkgs-unstable> {}}:
let
  inherit (nixpkgs) pkgs;
  python = pkgs.python36.withPackages (ps: with ps; [
    jupyter
    scikitlearn
    pandas
    numpy
    #matplotlib
    graphviz
    networkx
  ]);
in
  pkgs.stdenv.mkDerivation {
    name = "jupyter-env";
    buildInputs = [ 
      python 
      pkgs.python36Packages.jupyter_core
      pkgs.python36Packages.matplotlib
    ];
  }
