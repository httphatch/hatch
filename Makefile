VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/httphatch/hatch/cmd.version=$(VERSION) \
	-X github.com/httphatch/hatch/cmd.commit=$(COMMIT) \
	-X github.com/httphatch/hatch/cmd.date=$(DATE)

DEV_HOME := $(HOME)/.hatch-dev

.PHONY: build build-test run test lint clean app frontend icon docs dev dev-init dev-down dev-status

build: frontend
	go build -ldflags '$(LDFLAGS)' -o hatch .

build-test:
	go build -ldflags '$(LDFLAGS)' -o testing/hatch .

run:
	go run -ldflags '$(LDFLAGS)' . $(ARGS)

test:
	go test ./...

lint:
	golangci-lint run

frontend:
	cd frontend && npm ci && VITE_APP_VERSION=$(VERSION) npm run build

app: frontend
	go build -ldflags '$(LDFLAGS) -extldflags -w' -o hatch .

icon:
	rsvg-convert -f png -w 44 -h 44 split-solid-full.svg -o internal/tray/icon.png

docs:
	cd docs && npm ci && npm run dev

define DEV_CONFIG
version: 1
settings:
  tld: dev
  http_port: 8080
  https_port: 8443
  api_port: 42825
  dns_port: 5054
  caddy_admin_port: 2020
  auto_start: false
  tray_icon: false
  log_level: debug
projects: {}
endef
export DEV_CONFIG

dev-init: build
	HATCH_HOME=$(DEV_HOME) ./hatch init
	@if [ ! -f $(DEV_HOME)/.dev-configured ]; then \
		echo "Writing dev config with non-conflicting ports..."; \
		echo "$$DEV_CONFIG" > $(DEV_HOME)/config.yml; \
		touch $(DEV_HOME)/.dev-configured; \
	fi

dev: build
	HATCH_HOME=$(DEV_HOME) ./hatch up

dev-down:
	HATCH_HOME=$(DEV_HOME) ./hatch down

dev-status:
	HATCH_HOME=$(DEV_HOME) ./hatch status

clean:
	rm -f hatch
