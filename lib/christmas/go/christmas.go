package christmas

//go:generate mkdir -p christmaspb
//go:generate protoc -I=.. --go_out=paths=source_relative:./christmaspb christmas.proto
