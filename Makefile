GOPATH ?= $(shell go env GOPATH)
OS_NAME = $(shell uname)

.PHONY: test
test:
	go test -v ./... -p 1

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: lint
lint:
	@echo Installing linters...
	@test -e $(GOPATH)/bin/impi || \
		(mkdir -p $(GOPATH)/bin && \
		curl -s https://api.github.com/repos/pavius/impi/releases/latest \
		| grep -i "browser_download_url.*impi.*$(OS_NAME)" \
		| cut -d : -f 2,3 \
		| tr -d \" \
		| wget -O $(GOPATH)/bin/impi -qi - \
		&& chmod +x $(GOPATH)/bin/impi)

	@test -e $(GOPATH)/bin/golangci-lint || \
	  	(curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.41.1)

	@echo Verifying imports...
	$(GOPATH)/bin/impi \
		--local github.com/nuclio/gosecretive/ \
		--scheme stdLocalThirdParty \
		./...

	@echo Linting...
	$(GOPATH)/bin/golangci-lint run -v
	@echo Done.
