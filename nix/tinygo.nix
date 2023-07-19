{ lib, stdenv, fetchzip, go, runCommandLocal, makeWrapper }:

let
	version = "0.28.1";
	os = "linux";

	src = {
		amd64 = fetchzip {
			url = "https://github.com/tinygo-org/tinygo/releases/download/v0.28.1/tinygo${version}.${os}-amd64.tar.gz";
			sha256 = "sha256-6LxGiphcNxpm9uWbWKcPWBAVppsQoNs9RCHOuLm1E+o=";
		};
	};
in

runCommandLocal "tinygo-${version}" {
	inherit version;
	nativeBuildInputs = [ makeWrapper ];
} ''
	cp --no-preserve=mode,ownership -r ${src.amd64} $out
	chmod +x $out/bin/*
	wrapProgram $out/bin/tinygo \
		--set GOROOT ${go}/share/go \
		--set TINYGOROOT $out
''
