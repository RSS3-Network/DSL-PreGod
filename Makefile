VERSION=$(shell git rev-parse --short HEAD)

initialize:
	go work sync

.PHONY: build
build:
	mkdir -p ./build
	go build \
    	-ldflags "-w -s -X github.com/naturalselectionlabs/pregod/common/constant.Version=$(VERSION)" \
    	-o ./build/pregod_hub ./service/hub/cmd/main.go
	go build \
    	-ldflags "-w -s -X github.com/naturalselectionlabs/pregod/common/constant.Version=$(VERSION)" \
        -o ./build/pregod_indexer ./service/indexer/cmd/main.go
