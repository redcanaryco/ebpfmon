# ebpfmon
ebpfmon is a tool for monitoring eBPF programs. It is designed to be used with
the [bpftool](https://github.com/libbpf/bpftool) from the linux kernel. ebpfmon
is a TUI (terminal UI) application written in Go that allows you to do real-time
monitoring of eBPF programs.

# Installation
Right now the only supported way to install ebpfmon is to build it from source.

## Dependencies
First make sure to install the following dependencies. These are for bpftool to work.

Required dependencies for bpftool to work
- libelf
- zlib

Optional (but highly recommended) dependencies
- libcap
- libbfd
- clang/llvm

### Ubuntu
To install all the dependencies run the following command:
```bash
$ sudo apt install libelf-dev zlib1g-dev clang llvm binutils-dev
```

## Building
```bash
$ git clone --recurse-submodules https://github.com/redcanaryco/ebpfmon && cd ebpfmon
```

```bash
$ git submodule init --update --recursive
```

```bash
$ make
```

# Usage
```bash
$ ./ebpfmon
```

NOTE: `bpftool` needs root privileges and so ebpfmon will run `sudo bpftool ...`.
This means you will likely be prompted to enter your sudo password.