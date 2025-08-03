build-server:
	go build -o bin/deploy-server ./server

build-client:
	go build -o bin/go-deploy ./client

build: build-server build-client

run-server:
	./bin/deploy-server

clean:
	rm -rf bin/
	rm -rf uploads/

.PHONY: build-server build-client build run-server clean