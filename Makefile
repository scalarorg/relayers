PACKAGES=$(shell go list ./... | grep -v '/simulation')

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
HTTPS_GIT := https://github.com/scalarorg/relayers.git
PUSH_DOCKER_IMAGE := true

# Default values that can be overridden by the caller via `make VAR=value [target]`
# NOTE: Avoid adding comments on the same line as the variable assignment since trailing spaces will be included in the variable by make
WASM := true
WASMVM_VERSION := v2.1.3
# 3 MB max wasm bytecode size
MAX_WASM_SIZE := $(shell echo "$$((3 * 1024 * 1024))")
IBC_WASM_HOOKS := false
# Export env var to go build so Cosmos SDK can see it
export CGO_ENABLED := 1
# Add bitcoin-vault lib to CGO_LDFLAGS and CGO_CFLAGS
export CGO_LDFLAGS := ${CGO_LDFLAGS} -lbitcoin_vault_ffi


$(info $$WASM is [${WASM}])
$(info $$IBC_WASM_HOOKS is [${IBC_WASM_HOOKS}])
$(info $$MAX_WASM_SIZE is [${MAX_WASM_SIZE}])
$(info $$CGO_ENABLED is [${CGO_ENABLED}])
ifndef $(VERSION)
VERSION := "v0.0.1"
endif
ifndef $(WASM_CAPABILITIES)
# Wasm capabilities: https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
WASM_CAPABILITIES := "iterator,staking,stargate,cosmwasm_2_1"
else
WASM_CAPABILITIES := ""
endif

ifeq ($(MUSLC), true)
STATIC_LINK_FLAGS := -linkmode=external -extldflags '-Wl,-z,muldefs -static'
BUILD_TAGS := muslc,ledger
else
STATIC_LINK_FLAGS := ""
BUILD_TAGS := ledger
endif

ARCH := x86_64
ifeq ($(shell uname -m), arm64)
ARCH := aarch64
endif

ldflags = "-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(BUILD_TAGS)" \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-w -s ${STATIC_LINK_FLAGS}"

BUILD_FLAGS := -tags $(BUILD_TAGS) -ldflags $(ldflags) -trimpath
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)
OS := $(shell echo $$OS_TYPE | sed -e 's/ubuntu-20.04/linux/; s/macos-latest/darwin/')
SUFFIX := $(shell echo $$PLATFORM | sed 's/\//-/' | sed 's/\///')

.PHONY: all
all: goimports lint build docker-image docker-image-debug

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

# Uncomment when you have some tests
# test:
# 	@go test -mod=readonly $(PACKAGES)
.PHONY: lint
# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

# Build the project with release flags
.PHONY: build
build: go.sum
		go build -o ./bin/relayer -mod=readonly $(BUILD_FLAGS) ./main.go

.PHONY: build-binaries
build-binaries:  guard-SEMVER
	./scripts/build-binaries.sh ${SEMVER} '$(BUILD_TAGS)' '$(ldflags)'

# Build the project with release flags for multiarch
.PHONY: build-binaries-multiarch
build-binaries-multiarch: go.sum
		GOOS=${OS} GOARCH=${ARCH} go build -o ./bin/relayer -mod=readonly $(BUILD_FLAGS) ./main.go

.PHONY: build-binaries-in-docker
build-binaries-in-docker:  guard-SEMVER
	DOCKER_BUILDKIT=1 docker build \
		--build-arg SEMVER=${SEMVER} \
		-t scalarorg/relayer:binaries \
		-f Dockerfile.binaries .
	./scripts/copy-binaries-from-image.sh

# Build the project with debug flags
.PHONY: debug
debug:  go.sum
		go build -o ./bin/relayer -mod=readonly $(BUILD_FLAGS) -gcflags="all=-N -l" ./main.go

# Build a release image
.PHONY: docker-image
docker-image:
	@DOCKER_BUILDKIT=1 docker build \
		--build-arg WASM="${WASM}" \
		--build-arg WASMVM_VERSION="${WASMVM_VERSION}" \
		--build-arg IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" \
		--build-arg ARCH="${ARCH}" \
		-t scalarorg/relayer .

# Build a release image
.PHONY: docker-image-local-user
docker-image-local-user:  guard-VERSION guard-GROUP_ID guard-USER_ID
	@DOCKER_BUILDKIT=1 docker build \
		--build-arg USER_ID=${USER_ID} \
		--build-arg GROUP_ID=${GROUP_ID} \
		--build-arg WASM="${WASM}" \
		--build-arg IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" \
		--build-arg ARCH="${ARCH}" \
		-t scalarorg/relayer:${VERSION}-local .

.PHONY: build-push-docker-image
build-push-docker-images:  guard-SEMVER
	@DOCKER_BUILDKIT=1 docker buildx build \
		--platform ${PLATFORM} \
		--output "type=image,push=${PUSH_DOCKER_IMAGE}" \
		--build-arg WASM="${WASM}" \
		--build-arg IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" \
		--build-arg ARCH="${ARCH}" \
		-t scalarorg/relayer-${SUFFIX}:${SEMVER} --provenance=false .


.PHONY: build-push-docker-image-rosetta
build-push-docker-images-rosetta: populate-bytecode guard-SEMVER
	@DOCKER_BUILDKIT=1 docker buildx build -f Dockerfile.rosetta \
		--platform linux/amd64 \
		--output "type=image,push=${PUSH_DOCKER_IMAGE}" \
		--build-arg WASM="${WASM}" \
		--build-arg IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" \
		-t scalarorg/relayer:${SEMVER}-rosetta .


# Build a docker image that is able to run dlv and a debugger can be hooked up to
.PHONY: docker-image-debug
docker-image-debug:
	@DOCKER_BUILDKIT=1 docker build --build-arg WASM="${WASM}" --build-arg IBC_WASM_HOOKS="${IBC_WASM_HOOKS}" -t scalarorg/relayer -f ./Dockerfile.debug .


guard-%:
	@ if [ -z '${${*}}' ]; then echo 'Environment variable $* not set' && exit 1; fi
