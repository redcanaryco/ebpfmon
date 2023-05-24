// Generic utility functions used by other packages
package utils

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CgroupProcNotFoundError struct{}

func (m *CgroupProcNotFoundError) Error() string {
	return "No cgroup.procs file found or no processes in cgroup"
}

type CgroupProcess struct {
	// The pid of a process in a cgroup
	Pid int

	// The path to the cgroup the process is in
	CgroupPath string

	// The comm string of the process in the cgroup
	Comm string

	// The cmdline of the process in the cgroup
	Cmdline string

	// The path of the executable of the process in the cgroup
	Path string
}

// Get the name of a process using the pid
func GetProcessName(pid int) (string, error) {
	cmd := exec.Command("sudo", "ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Convert the output to a string
	outStr := string(out)
	// Trim the newline
	outStr = strings.TrimSuffix(outStr, "\n")
	return outStr, nil
}

// Get the path of a process using the pid
func GetProcessPath(pid int) (string, error) {
	cmd := exec.Command("sudo", "readlink", "-f", "/proc/" + strconv.Itoa(pid) + "/exe")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Convert the output to a string
	outStr := string(out)
	// Trim the newline
	outStr = strings.TrimSuffix(outStr, "\n")
	return outStr, nil
}

// Get the processes inside of a given cgroup
func GetProcsInCgroup(cgroupPath string) ([]CgroupProcess, error) {
	var result []CgroupProcess

	// Read the tasks file to get the list of processes
	tasks, err := os.ReadFile(filepath.Join(cgroupPath, "cgroup.procs"))
	if err != nil {
		return []CgroupProcess{}, err
	}

	// Convert the tasks to a list of strings
	processes := strings.Split(strings.TrimSpace(string(tasks)), "\n")


	if len(processes) == 1 && processes[0] == "" {
		return []CgroupProcess{}, &CgroupProcNotFoundError{}
	}

	for _, procId := range processes {
		procId, err := strconv.Atoi(procId)
		if err != nil {
			return []CgroupProcess{}, err
		}
		comm, err := GetProcessName(procId)
		if err != nil {
			comm = ""
		}
		cmdline, err := GetProcessCmdline(procId)
		if err != nil {
			cmdline = ""
		}
		path, err := GetProcessPath(procId)
		if err != nil {
			path = ""
		}
		result = append(result, CgroupProcess {
			Pid: procId,
			CgroupPath: cgroupPath,
			Comm: comm,
			Cmdline: cmdline,
			Path:path,
		})
	}
	return result, nil
}

// Get a list of processes in a cgroup. The cgroup path is optional and defaults
// to /sys/fs/cgroup
func parseCgroups(path ...string) (map[string][]string, error) {
	cgroups := make(map[string][]string)

	// Walk the cgroups directory recursively
	cgroupPath := "/sys/fs/cgroup"
	if len(path) == 1 {
		cgroupPath = path[0]
	}
	err := filepath.Walk(cgroupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process directories with a "tasks" file
		if info.IsDir() && FileExists(filepath.Join(path, "cgroup.procs")) {
			// Read the tasks file to get the list of processes
			tasks, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
			if err != nil {
				return err
			}

			// Convert the tasks to a list of strings
			processes := strings.Split(strings.TrimSpace(string(tasks)), "\n")

            if len(processes) == 1 && processes[0] == "" {
                return nil
            }

			// Build the cgroup path by removing the "/sys/fs/cgroup/" prefix from the path
			cgroup := strings.TrimPrefix(path, "/sys/fs/cgroup/")

			// Add the cgroup and its processes to the map
			cgroups[cgroup] = processes
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return cgroups, nil
}

// Get the command line of a process using the pid
func GetProcessCmdline(procId int) (string, error) {
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", procId))
	if err != nil {
		return "", err
	}
	cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
	return cmdline, err
}

func isNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

// Check if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Simple wrapper for running a command and returning the stdout and stderr
func RunCmd(args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func FindProcByProgId(progId int) ([]int, error) {
	var pids []int
	id := strconv.Itoa(progId)


	// Walk the /proc directory
	err := filepath.WalkDir("/proc", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Only process directories with a numeric name (i.e., a process ID)
		if info.IsDir() && isNumeric(info.Name()) {
			pid, err := strconv.Atoi(info.Name())
			if err != nil {
                // fmt.Printf("Failed to convert %s\n", info.Name())
				return nil
			}

			// Read the files in the /proc/<pid>/fdinfo directory
			fdinfoDir := filepath.Join(path, "fdinfo")
			if FileExists(fdinfoDir) && !strings.Contains(fdinfoDir, "task") {
				filesSlice := []string{}
				files, err := os.ReadDir(fdinfoDir)
				for _, file := range files {
					filesSlice = append(filesSlice, filepath.Join(fdinfoDir, file.Name()))
				}

				if err != nil {
                    // fmt.Printf("Failed to read%s\n", fdinfoDir)
					return nil
				}

				// Check each file for the prog_id string
				for _, file := range filesSlice {
					// fmt.(file)

					content, err := os.ReadFile(file)
					if err != nil {
						// fmt.Printf("Failed to read fdinfo file %s\n", file)
						continue
					}

					// Write a regex for detecting the prog_id string
					// The pattern looks like this:
					// prog_id:       1
					r := regexp.MustCompile(`prog_id:\s+\d+`)
					match := r.FindString(string(content))
					if match != "" {
						curProgid := strings.TrimSpace(strings.Split(match, ":")[1])
						if curProgid == id {
							pids = append(pids, pid)
							break
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
        // fmt.Println("Failed")
		return nil, err
	}

	return pids, nil
}

// Write a function that checks if an int is in a slice
func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Write a function with a receiver to compare two BpfProgram structs
func (p BpfProgram) Compare(other BpfProgram) bool {
	if p.ProgramId == other.ProgramId && 
	   p.Tag == other.Tag {
		return true
	}
	return false
}

// Tview uses a special syntax in strings to colorize things. This function
// removes those color codes from a string
// And example string would look like "[blue]mystring[-]"
func RemoveStringColors(s string) string {
	if s == "" {
		return s
	}
	
	// Write a regex to match the color codes
	r := regexp.MustCompile(`\[\w+\]|\[-\]`)
	// Replace the color codes with an empty string
	result := r.ReplaceAllString(s, "")
	return result
}

func Which(program string) (string, error) {
	path, err := exec.LookPath("bpftool")
	if err != nil {
		fmt.Println("Failed to find compiled version of bpftool")
		return "", err
	} else {
		path, err = filepath.Abs(path)
		if err != nil {
			fmt.Println("Failed to find compiled version of bpftool")
			return "", err
		}
	}
	return path, nil
}