{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = with pkgs; [
		ffmpeg-full
		go
		bash
	];

	shellHook = ''
		export PATH="$PATH:${builtins.toString ./.}/bin"
	'';
}
