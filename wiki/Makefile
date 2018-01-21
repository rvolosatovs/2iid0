all: bin/wiki-linux bin/wiki-windows

bin/wiki-linux: graph.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o bin/wiki-linux graph.go 

bin/wiki-windows: graph.go
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o bin/wiki-windows graph.go 
