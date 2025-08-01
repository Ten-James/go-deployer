build-server:
	go build -o deploy-server server.go

build-client:
	go build -o go-deploy client.go

build: build-server build-client

run-server:
	./deploy-server

clean:
	rm -f deploy-server go-deploy
	rm -rf uploads/

.PHONY: build-server build-client build run-server clean