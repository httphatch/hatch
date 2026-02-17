VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/httphatch/hatch/cmd.version=$(VERSION) \
	-X github.com/httphatch/hatch/cmd.commit=$(COMMIT) \
	-X github.com/httphatch/hatch/cmd.date=$(DATE)

.PHONY: build build-test run test lint clean app frontend icon docs

build:
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
	cd frontend && npm install && VITE_APP_VERSION=$(VERSION) npm run build

app: frontend
	go build -ldflags '$(LDFLAGS) -extldflags -w' -o hatch .

icon:
	rsvg-convert -f png -w 44 -h 44 split-solid-full.svg -o internal/tray/icon.png

docs:
	cd docs && npx vitepress dev

clean:
	rm -f hatch
