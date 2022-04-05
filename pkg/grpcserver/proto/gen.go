package proto

//go:generate protoc --go_opt=paths=source_relative --go_out=plugins=grpc:. helloworld/v1/helloworld.proto
//go:generate protoc --go_opt=paths=source_relative --go_out=plugins=grpc:. health/v1/health.proto
