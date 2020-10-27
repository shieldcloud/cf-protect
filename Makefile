PLUGIN := protect

test:
	go test .

LDFLAGS := -X main.Version=$(VERSION)
release:
	@echo "Checking that VERSION was defined in the calling environment"
	@test -n "$(VERSION)"
	@echo "OK.  VERSION=$(VERSION)"
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o cf-protect.linux64
	GOOS=linux   GOARCH=386   go build -ldflags="$(LDFLAGS)" -o cf-protect.linux32
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o cf-protect.darwin64
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o cf-protect.win64
	GOOS=windows GOARCH=386   go build -ldflags="$(LDFLAGS)" -o cf-protect.win32

install:
	go build .
	cf uninstall-plugin $(PLUGIN) || true
	yes | cf install-plugin cf-$(PLUGIN)

.PHONY: test release install
