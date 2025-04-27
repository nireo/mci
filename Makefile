PROTO_DIR := pb
GEN_DIR := pb

GO_PACKAGE := pb

PROTOS := $(shell find $(PROTO_DIR) -name '*.proto')

PROTOC_CMD := protoc \
	--proto_path=$(PROTO_DIR) \
	--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative

.PHONY: gen
gen: $(GEN_DIR) $(PROTOS)
	@echo "Generating Go code from .proto files..."
	$(PROTOC_CMD) $(PROTOS)
	@echo "Done."

$(GEN_DIR):
	@echo "Creating output directory: $(GEN_DIR)"
	@mkdir -p $(GEN_DIR)

.PHONY: clean
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(GEN_DIR)
	@echo "Done."

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make gen    - Generate Go gRPC code from .proto files"
	@echo "  make clean  - Remove generated Go code"
	@echo "  make help   - Show this help message"

.PHONY: test
test:
	go test -v ./...
