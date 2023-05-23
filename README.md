# ebpfmon
ebpfmon is a tool for monitoring eBPF programs. It is designed to be used with
the [bpftool](https://github.com/libbpf/bpftool) from the linux kernel. ebpfmon
is a TUI (terminal UI) application written in Go that allows you to do real-time
monitoring of eBPF programs.

# Installation
Right now the only supported way to install ebpfmon is to build it from source.

## Dependencies
First and foremost this tool is written in [Go](https://go.dev/learn/) so you will need to have that installed and in your PATH variable. It should work on go 1.18 or later although it's possible it could work on earlier versions. It just hasn't been tested

Next make sure to install the following dependencies. These are for bpftool to work.

Required dependencies for bpftool to work
- bpftool (installed from a package manager or build from source). This is what ebpfmon uses to get information regarding eBPF programs, maps, etc
- libelf
- zlib

Optional (but highly recommended) dependencies
- libcap-devel
- libbfd

Optional dependencies for additional features
- clang/llvm

### Ubuntu 20.04+
To install all the dependencies run the following command:
```bash
$ sudo apt install linux-tools-`uname -r` libelf-dev zlib1g-dev libcap-dev clang llvm binutils-dev
```

### Amazon Linux 2
To install all the dependencies run the following command:
```bash
$ sudo yum install elfutils-libelf-devel libcap-devel binutils-devel clang bpftool
```

### Rhel, CentOS, Fedora
To install all the dependencies run the following command:
```bash
$ sudo dnf install elfutils-libelf-devel libcap-devel zlib-devel binutils-devel clang bpftool
```

### Debian 11 
```bash
$ sudo apt install bpftool libelf-dev zlib1g-dev libcap-dev binutils-dev clang llvm  
```


## Building
```bash
$ git clone --recurse-submodules https://github.com/redcanaryco/ebpfmon
$ cd ebpfmon
```

or

```bash
$ git clone https://github.com/redcanaryco/ebpfmon
$ cd ebpfmon
$ git submodule update --init --recursive
```

Then simply run. This will build the `ebpfmon` binary in the current directory
```bash
$ make
```

# Usage
```bash
$ ./ebpfmon
```

NOTE: `bpftool` needs root privileges and so ebpfmon will run `sudo bpftool ...`.
This means you will likely be prompted to enter your sudo password.

# Documentation
## Command Line Arguments
### `-bpftool`
Allows you to specify the path to the bpftool binary. This is useful if you have
a custom build of bpftool that you want to use. By default it will use the
system's bpftool binary. You can also use an environemnt variable. It will look
in the following order
1. Check if the `-bpftool` argument was speified on the command line
2. Check if the environment variable `BPFTOOL_PATH` is set.
3. Use the system binary 

### `-logfile`
This argument allows you to specify a file to log to. By default it will log to
`./log.txt`. This is a great file to check when trying to debug issues with the
application as it will log errors that occured during runtime.