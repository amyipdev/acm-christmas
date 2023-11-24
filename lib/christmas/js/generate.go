package ignore

//go:generate nix-shell --run "protoc -I=.. --plugin=./node_modules/.bin/protoc-gen-ts_proto --ts_proto_opt=esModuleInterop=true --ts_proto_opt=importSuffix=.js --ts_proto_out=src/christmaspb/ christmas.proto"
//go:generate nix-shell --run "esbuild --bundle src/index.ts --outdir=. --minify --sourcemap"
//go:generate nix-shell --run "tsc --emitDeclarationOnly --outFile index.d.ts  --project ./tsconfig.json"
