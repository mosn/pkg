BUILD_IMAGE     = godep-builder
SCRIPT_DIR      = $(shell pwd)/etc/script
PROJECT_NAME    = mosn.io/pkg

coverage-local:
	sh ${SCRIPT_DIR}/report.sh

coverage:
	docker build --rm -t ${BUILD_IMAGE} build/contrib/builder/binary
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make coverage-local
