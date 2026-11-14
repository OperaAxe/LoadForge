.PHONY: test vet build dashboard clean

test:
	go test ./...

vet:
	go vet ./...

dashboard:
	cd web && npm install && npm run build

build: dashboard
	go build -o bin/loadforge ./cmd/loadforge

clean:
	rm -rf bin web/dist
