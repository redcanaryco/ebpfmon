package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var BpftoolPath string

type ProcessInfo struct {
	Pid int `json:"pid"`
	Comm string `json:"comm"`
	Cmdline string
	Path string
	Uid int
	Gid int
}

type BpfMap struct {
	// The id of the map
	Id int `json:"id"`

	// The type of map
	Type string `json:"type"`

	// The name of the map if present
	Name string `json:"name",omitempty`

	// Any flags that are set on the map
	Flags int `json:"flags"`

	// The key size (in bytes)
	KeySize int `json:"bytes_key"`

	// The value size (in bytes)
	ValueSize int `json:"bytes_value"`

	// The max number of entries in the map
	MaxEntries int `json:"max_entries"`

	// The amount of memory the map lockes in
	Memlock int `json:"bytes_memlock"`

	// The btf id referenced by the map
	BtfId int `json:"btf_id",omitempty`

	// The state of the map. Examples could be frozen, pinned etc
	Frozen int  `json:"frozen",omitempty`

	// If the map is pinned the path will be here
	Pinned []string `json:"pinned",omitempty`
}

type BpfMapEntryRaw struct {
	// The key of the map entry
	Key []string `json:"key"`

	// The value of the map entry
	Value []string `json:"value"`

	// The formatted value of the map entry if it exists
	Formatted struct {
		// Value can be a variety of things
		Value interface{} `json:"value"`
	} `json:"formatted",omitempty`
}

type BpfMapEntry struct {
	// The key of the map entry
	Key []byte `json:"key"`

	// The value of the map entry
	Value []byte `json:"value"`

	// The formatted value of the map entry if it exists
	Formatted struct {
		// Value can be a variety of things
		Value interface{} `json:"value"`
	} `json:"formatted",omitempty`
}

type BpfProgram struct {
	// The name of the program. This field may be empty
    Name string `json:"name",omitempty`

	// The tag of the program. This field should not be empty
    Tag string `json:"tag"`

	// The bpf program id
    ProgramId int `json:"id"`

	// The type of the program i.e. Kprobe, Kretprobe, Uprobe etc
    ProgType string `json:"type"`

	// The license of the program. May be empty
    GplCompatible bool `json:"gpl_compatible"`

	// The time the program was loaded
    LoadedAt int `json:"loaded_at"`

	// The uid of the owner
	OwnerUid int `json:"uid"`

	// The number of instructions in the xlated version of the program
	BytesXlated int `json:"bytes_xlated"`

	// Whether or not the program is jited
	Jited bool `json:"jited"`

	// The number of jited instructions
	BytesJited int `json:"bytes_jited"`

	// The amount of memory that is locked
	BytesMemlock int `json:"bytes_memlock"`

	// The ids of any maps the program references
    MapIds []int `json:"map_ids",omitempty`

	// The id of an btf objects the program references
    BtfId int `json:"btf_id",omitempty`

	// If the program is pinned this field will contain the path
	Pinned []string `json:"pinned",omitempty`

	Pids []ProcessInfo `json:"pids"`

	// The attach points of the program. There may be multiple
	AttachPoint []string

	// The offset from the attach point
	Offset int

	// The fd of the bpf program
	Fd int

	// A sha256 hash representing a unique id for the program
	Fingerprint []string

	// The disassembly of the program
	Instructions []string

	// The network interface this program is attached to
	Interface string

	// The type of TC program
	TcKind string

	// Cgroup that the program is attached to
	Cgroup string

	// The attach type for the cgroup. Examples are ingress, egress, device, bind4, bind6 etc
	CgroupAttachType string

	// Either multi or override
	CgroupAttachFlags string

}

// Write a stringer for BpfProgram
func (p BpfProgram) String() string {
	result := fmt.Sprintf("%6d: [green]%13s[-] [blue]%16s[-] %20s ", p.ProgramId, p.ProgType, p.Tag, p.Name)
	for _, point := range p.AttachPoint {
		result += fmt.Sprintf("%s, ", point)
	}
	result = strings.TrimSuffix(result, ", ")
	return result
}

// Write a stringer for BpfMap
func (p BpfMap) String() string {
	if p.Name == "" {
		return fmt.Sprintf("%6d: %s", p.Id, p.Type)
	} else {
		return fmt.Sprintf("%6d: %s %s", p.Id, p.Type, p.Name)
	}
}

// Write a stringer for BpfMapEntry
func (e BpfMapEntry) String() string {
	result := fmt.Sprintf("%v: %v", e.Key, e.Value)
	return result
}

// Call the bpftool binary to get the disassembly of a program
// using the program id
func GetBpfProgramDisassembly(programId int) ([]string, error) {
	stdout, _, err := RunCmd("sudo", BpftoolPath, "prog", "dump", "xlated", "id", strconv.Itoa(programId))
	if err != nil {
		return []string{}, err
	}
	// Convert the output to a string
	outStr := string(stdout)
	// Split the output into lines
	result := strings.Split(outStr, "\n")
	return result, nil
}

// Use bpftool map show to get the map info
func GetBpfMapInfo() ([]BpfMap, error) {
	var bpfMap []BpfMap
	stdout, stderr, err := RunCmd("sudo", BpftoolPath, "map", "-jf", "show") 
	if err != nil {
		fmt.Printf("Error running bpftool map: %s\n", err)
		fmt.Printf("Stderr: %s\n", stderr)
		return bpfMap, err
	}

	err = json.Unmarshal(stdout, &bpfMap)
	if err != nil {
		fmt.Printf("Error parsing bpftool map output: %s\n", err)
		return bpfMap, err
	}
	return bpfMap, nil
}

// Parse the output of the bpftool binary to get the map info that correspond
// to the map ids the bpf program is using
func GetBpfMapInfoByIds(mapIds []int) []BpfMap {
	tmp := []BpfMap{}
	result := []BpfMap{}
	
	// Call the bpftool binary to get the map info
	stdout, stderr, err := RunCmd("sudo", BpftoolPath, "-j", "map", "show")
	if err != nil {
		fmt.Printf("Failed to run `sudo %s -j map show`\n", BpftoolPath)
		fmt.Println(string(stderr))
		panic(err)
	}
	err = json.Unmarshal(stdout, &tmp)
	if err != nil {
		panic(err)
	}

	for _, m := range tmp {
		if contains(mapIds, m.Id) {
			result = append(result, m)
		}
	}
	return result
}

func convertStringSliceToByteSlice(strSlice []string) ([]byte, error) {

	byteSlice := make([]byte, len(strSlice))

	for i, str := range strSlice {
		// Remove the "0x" prefix from the string
		if len(str) >= 2 && str[:2] == "0x" {
			str = str[2:]
		}

		// Parse the string as a hexadecimal value
		bytes, err := hex.DecodeString(str)
		if err != nil {
			fmt.Printf("Error decoding hexadecimal value: %s\n", err)
			return []byte{}, err
		}

		// Append the byte value to the byte slice
		byteSlice[i] = bytes[0]
	}

	return byteSlice, nil
}

// Use bpftool map dump to get the data from a map
// TODO: Do we want to work with the raw string values or a byte value?
// TODO: e.g. ["0x00", "0x01", "0x02", ...] or [0x00, 0x01, 0x02, ...]?
func GetBpfMapEntries(mapId int) []BpfMapEntry {
	var result []BpfMapEntry
	var mapData []BpfMapEntryRaw
	stdout, _, err := RunCmd("sudo", BpftoolPath, "map", "-jf", "dump", "id", strconv.Itoa(mapId))
	if err != nil {
		// TODO: Whats the right way to indicate this to the user?
		return result
	}

	// Convert map data to individual elements
	err = json.Unmarshal(stdout, &mapData)

	// Use hex.DecodeString to convert the key and value to byte slices
	for i, _ := range mapData {
		b, err := convertStringSliceToByteSlice(mapData[i].Key)
		if err != nil {
			fmt.Printf("Error decoding key: %s\n", err)
		}

		v, err := convertStringSliceToByteSlice(mapData[i].Value)
		if err != nil {
			fmt.Printf("Error decoding value: %s\n", err)
		}

		result = append(result, BpfMapEntry{Key: b, Value: v})
	}

	return result
}
