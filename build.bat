SET BINARY_NAME=ela

go build -o bin/%BINARY_NAME%.exe

SET CGO_ENABLED=0
SET GOOS=darwin
SET GOARCH=amd64
go build -o bin/%BINARY_NAME%.darwin.amd64


SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o bin/%BINARY_NAME%.linux.amd64
