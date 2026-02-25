PROTO_OUT=sdk/go/proto
BUF_VERSION=1.32.1
BUF_BIN=bin/buf

all: generate

generate: $(BUF_BIN)
	$(BUF_BIN) generate

$(BUF_BIN):
	@mkdir -p bin
	@echo "Installing buf $(BUF_VERSION) locally..."
	@BUF_URL="https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$$(uname -s)-$$(uname -m)"; \
	curl -fsSL "$$BUF_URL" -o $(BUF_BIN) || { \
		echo "Failed to download buf from $$BUF_URL"; \
		exit 1; \
	}
	@chmod +x $(BUF_BIN)
	@if ! file $(BUF_BIN) | grep -q 'ELF\|Mach-O'; then \
		echo "Error: Downloaded file is not a valid binary"; \
		head -5 $(BUF_BIN); \
		rm -f $(BUF_BIN); \
		exit 1; \
	fi
	@echo "buf installed to $(BUF_BIN)"

tidy:
	go mod tidy

clean:
	rm -rf $(PROTO_OUT)

clean-all: clean
	rm -rf bin

test:
	go test ./...

buf: $(BUF_BIN)

buf-lint: $(BUF_BIN)
	$(BUF_BIN) lint

lint: $(BUF_BIN) lint-go lint-markdown lint-yaml

lint-go: $(BUF_BIN)
	PATH=~/go/bin:$$PATH golangci-lint run
	$(BUF_BIN) lint

lint-markdown:
	npm run lint:markdown

lint-markdown-fix:
	npm run lint:markdown:fix

lint-yaml:
	yamllint .github/

lint-yaml-fix:
	yamllint --fix .github/

validate-schema:
	npm run validate:schema

validate-examples:
	npm run validate:examples

validate-npm:
	npm run validate

validate: test lint validate-npm

depend: install-lefthook
	@echo "Installing Go development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.9.0
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/fatih/gomodifytags@latest
	@go install github.com/josharian/impl@latest
	@go install github.com/cweill/gotests/gotests@latest
	@go install github.com/golang/mock/mockgen@latest
	@go install github.com/axw/gocov/gocov@latest
	@go install github.com/AlekSi/gocov-xml@latest
	@go install github.com/tebeka/go2xunit@latest
	@echo "Go development tools installed successfully!"

# Lefthook git hooks management
.PHONY: install-lefthook uninstall-lefthook commitlint validate-commit

install-lefthook:
	@echo "Installing Lefthook..."
	@go install github.com/evilmartians/lefthook@latest
	@lefthook install
	@echo "Git hooks installed successfully!"

uninstall-lefthook:
	@echo "Uninstalling Lefthook git hooks..."
	@lefthook uninstall
	@echo "Git hooks uninstalled."

commitlint:
	@npx commitlint --from HEAD~1

validate-commit:
	@if [ -f PR_MESSAGE.md ]; then \
		echo "Validating PR commit message..."; \
		npx commitlint --edit PR_MESSAGE.md; \
	else \
		echo "Validating last commit message..."; \
		git log -1 --pretty=%B | npx commitlint; \
	fi