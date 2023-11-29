{ pkgs ? import <nixpkgs> {} }:

with pkgs;
mkShell {
	buildInputs = with pkgs; [
		protobuf
		python3
		python3Packages.black
		pyright
	];

	shellHook = ''
		python3 -m venv .venv
		source .venv/bin/activate
	'';
}
