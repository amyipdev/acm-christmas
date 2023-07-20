{}:

let
	src = import ./nix/sources.nix {};
	pkgs = import src.nixpkgs {};
in

let
	# Tinygo target for gopls to use.
	tinygoTarget = "esp32-coreboard-v2";
	tinygoPaths = [ "esp32" ];

	tinygoHook =
		with pkgs.lib;
		with builtins;
		''
			isTinygo() {
				root=${escapeShellArg (toString ./.)}
				path="''${PWD#"$root/"*}"

				for p in $TINYGO_PATHS; do
					if [[ $path == $p* ]]; then
						return 0
					fi
				done

				return 1
			}

			hookTinygoEnv() {
				vars=$(tinygo info -json -target $TINYGO_TARGET)

				export GOROOT=$(jq -r '.goroot' <<< "$vars")
				export GOARCH=$(jq -r '.goarch' <<< "$vars")
				export GOOS=$(jq -r '.goos' <<< "$vars")
				export GOFLAGS="-tags=$(jq -r '.build_tags | join(",")' <<< "$vars")"
			}
		'';

	withTinygoHook = name: bin:
		pkgs.writeShellScriptBin name ''
			${tinygoHook}
			if isTinygo; then
				echo "Detected Tinygo, loading for target $TINYGO_TARGET" >&2
				hookTinygoEnv
			fi
			exec ${bin} "$@"
		'';

  go = withTinygoHook "go" "${pkgs.go}/bin/go";
	gopls = withTinygoHook "gopls" "${pkgs.gopls}/bin/gopls";
	goimports = withTinygoHook "goimports" "${pkgs.gotools}/bin/goimports";

	staticcheck = pkgs.writeShellScriptBin "staticcheck" ''
		${tinygoHook}
		if isTinygo; then
			echo "Not running staticcheck for Tinygo" >&2
			exit 0
		fi
		exec ${pkgs.go-tools}/bin/staticcheck "$@"
	'';

	# Use the precompiled Tinygo which has ESP32 support.
	tinygo = pkgs.callPackage ./nix/tinygo.nix {};
in

with pkgs.lib;
with builtins;

pkgs.mkShell {
	buildInputs = with pkgs; [
		niv

		ffmpeg-full
		bash
		protobuf
		protolint

		# Go tools.
		go
		gopls
		goimports
		staticcheck

		# ESP32 programming.
		tinygo
		esptool
	];

	shellHook = ''
		export PATH="$PATH:${toString ./.}/bin"
	'';

	CGO_ENABLED = "1";
	TINYGO_PATHS = toString tinygoPaths;
	TINYGO_TARGET = tinygoTarget;
}
