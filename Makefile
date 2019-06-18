stubs:
	protoc -I proto proto/echoIP.proto --go_out=plugins=grpc:proto

server:
	go run server/server.go

server-with-pprof:
	go run server/server_with_pprof.go

client:
	go run client/client.go

graphviz:
	sudo apt install graphviz

cpu:
	go tool pprof --pdf cpu.prof > cpu.pdf

mem:
	go tool pprof --pdf -alloc_space mem.prof > mem.pdf

bench:
	go test -cpuprofile cpu.prof -memprofile mem.prof -bench .

ui-cpu:
	go tool pprof -http=:8080 cpu.prof

ui-mem:
	go tool pprof -http=:8080 mem.prof

build:
	go build -o qt_grpc mainUi.go
	chmod +x qt_grpc

gen_key:
	openssl req -newkey rsa:2048 -nodes -subj '/O=ASTRI/C=CN/OU=MSA/CN=localhost' \
	-keyout server/secret/server.key -x509 -days 365 -out server/secret/server.crt
