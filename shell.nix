{}:

let
	src = import ./nix/sources.nix {};
	pkgs = import src.nixpkgs {};
in

let
	# Tinygo target for gopls to use.
	tinygoTarget = "esp32-coreboard-v2";
	# Use all directories in cmd/ that start with "esp32-" for Tinygo.
	tinygoPaths =
		with builtins;
		with pkgs.lib;
		attrNames
			(filterAttrs
				(k: v: v == "directory" && hasPrefix "esp32-" k)
				(readDir (./. + "/cmd")));

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

	gopls =
		with pkgs.lib;
		with builtins;
		pkgs.writeShellScriptBin "gopls" ''
			${tinygoHook}
			if isTinygo; then
				echo "Detected Tinygo, loading for target $TINYGO_TARGET" >&2
				hookTinygoEnv
			fi
			exec ${pkgs.gopls}/bin/gopls "$@"
		'';
	
	# Disable staticcheck.
	staticcheck = pkgs.writeShellScriptBin "staticcheck" ''
		echo "staticcheck is disabled" >&2
	'';

	# Hijack Go to load Tinygo's environment if we're in a Tinygo project.
	go = pkgs.writeShellScriptBin "go" ''
		${tinygoHook}
		if isTinygo; then
			hookTinygoEnv
		fi
		exec ${pkgs.go}/bin/go "$@"
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
