CC = clang
CFLAGS = -Wall -g
OUTPUT = $(abspath ./.output)
BPFTOOL_SRC = $(abspath ./bpftool/src)

all: tool ebpfmon

tool:
	mkdir -p $(OUTPUT)
	$(MAKE) -C $(BPFTOOL_SRC)
	cp $(BPFTOOL_SRC)/bpftool $(OUTPUT)/bpftool

ebpfmon: main.go
	go build .

# Just cleans the binary
clean:
	rm -rf ebpfmon

# Only rebuild bpftool source if you really mean to
realclean: clean
	rm -rf $(OUTPUT)
	$(MAKE) -C $(BPFTOOL_SRC) clean

.PHONY: clean realclean