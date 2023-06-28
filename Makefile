proto: 
	@cd protocol && protoc --go_out=. --go_opt=paths=source_relative    --go-grpc_out=. --go-grpc_opt=paths=source_relative test.proto

docker-client:
	docker build --platform linux/amd64 -f Dockerfile.client -t gesellschaft-hale-client .\

docker-server:
	docker build --platform linux/amd64 -f Dockerfile.server -t gesellschaft-hale-server .\


docker-all: docker-client docker-server 

docker-client-push:
	docker tag gesellschaft-hale-client wotlk888/gesellschaft-hale:client 
	docker push wotlk888/gesellschaft-hale:client

docker-server-push:
	docker tag gesellschaft-hale-server wotlk888/gesellschaft-hale:server
	docker push wotlk888/gesellschaft-hale:server

docker-all-push: docker-client-push docker-server-push 

docker-client-run: 
	docker run -P -it --rm gesellschaft-hale-client 

docker-server-run:
	docker run -p 443:443 -p 50001:50001 -it --rm gesellschaft-hale-server
