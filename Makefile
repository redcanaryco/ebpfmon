all: ebpfmon

ebpfmon:
	go build .

# Just deletes the binary
clean:
	rm -rf ebpfmon

.PHONY: clean ebpfmon