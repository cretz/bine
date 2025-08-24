module github.com/cretz/bine/examples

go 1.23.0

require (
	github.com/cretz/bine v0.2.0
	github.com/golang/protobuf v1.5.2
	golang.org/x/net v0.43.0
	google.golang.org/grpc v1.38.0
)

require (
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)

replace github.com/cretz/bine => ../
