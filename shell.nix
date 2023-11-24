{}:

let
	pkgs = import ./nix/pkgs.nix {};
in

with pkgs.lib;
with builtins;

let
	gogio = pkgs.buildGoModule rec {
		pname = "gogio";
		version = "7cb98d05";

		src = pkgs.fetchgit {
			url = "https://git.sr.ht/~eliasnaur/gio-cmd";
			rev = version;
			sha256 = "sha256-sCNmTSBdg5CG2zdydd83OFjffIshtfEAIVLuHBXIckk=";
		};

		vendorSha256 = "sha256-2LQCFYyEletx+FswLV1Ui506qG62yHUKGr5vP5Y/b/s=";

		doCheck = false;
		subPackages = [ "gogio" ];
	};
in

pkgs.mkShell {
	buildInputs = with pkgs; [
		bash
		niv
		jq
		moreutils # for parallel
		ffmpeg-full

		# Protobuf tools.
		protobuf
		protolint
		protoc-gen-go

		# Go tools.
		go
		gopls
		gotools
		go-tools # staticcheck

		# WebAssembly.
		tinygo
	];

	shellHook = ''
		export PATH="$PATH:${toString ./.}/bin"
	'';

	CGO_ENABLED = "1";
}
