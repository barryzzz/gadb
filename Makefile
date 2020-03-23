BINARY_NAME=gadb
VERSION=1.0.3
build:

	rm -rf *.gz
	go clean
	go build -o $(BINARY_NAME)
	tar czvf gadb-mac64-${VERSION}.tar.gz ./${BINARY_NAME}
	go clean 
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ${BINARY_NAME}.exe
	tar czvf gadb-win64-${VERSION}.tar.gz ./${BINARY_NAME}.exe
	go clean
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME}
	tar czvf gadb-linux64-${VERSION}.tar.gz ./${BINARY_NAME}
	rm ${BINARY_NAME}

clean:
	rm -rf *.gz
	rm ${BINARY_NAME} ${BINARY_NAME}.exe
	