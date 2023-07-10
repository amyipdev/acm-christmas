{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
	buildInputs = with pkgs; [
		ffmpeg-full
		go
		bash
		protobuf
		protolint
	];

	shellHook = ''
		export PATH="$PATH:${builtins.toString ./.}/bin"
	'';
}
