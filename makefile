TAGS = linux

build-windows: tags-windows build # сборка windows

build:
	@echo "Build app..." ; \
	CGO_ENABLED=0 GOOS=$(TAGS) GOARCH=amd64 go build -ldflags "-s -w" -tags $(TAGS)

analysis:
	go vet ./$(FILEPATH)...
	staticcheck ./$(FILEPATH)...
	gosec ./$(FILEPATH)...
	ineffassign ./$(FILEPATH)*

tags-windows: # Установка тега для сборки под windows
	$(eval TAGS = windows)
	@echo "tags = $(TAGS)"
