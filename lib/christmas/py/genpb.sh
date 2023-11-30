set -e
cd "$(dirname "$0")"
protoc -I=.. --python_out=./acm_christmas ../christmas.proto