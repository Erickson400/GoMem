import (
	"encoding/binary"
	"fmt"
	win "golang.org/x/sys/windows"
	"strings"
	"unsafe"
	"math"
)

type Module struct {
	Name        string
	ModBaseAddr uintptr
	ModBaseSize uint32
}

type Process struct {
	Name        string
	Handle      win.Handle
	Pid         uint32
	ModBaseAddr uintptr
	ModBaseSize uint32
	Modules     map[string]Module
	BigEndian   bool
}

func processInfo(pid uint32) (Process, error) {
	snap, err := win.CreateToolhelp32Snapshot(win.TH32CS_SNAPMODULE32|win.TH32CS_SNAPMODULE, pid)
	if err != nil {
		return Process{}, err
	}

	var me win.ModuleEntry32
	me.Size = uint32(unsafe.Sizeof(me))
	err = win.Module32First(snap, &me)
	if err != nil {
		win.CloseHandle(snap)
		return Process{}, fmt.Errorf("ERROR: ProcessInfo failed on %d", pid)
	}

	proc := Process{
		Name:        win.UTF16PtrToString(&me.Module[0]),
		Pid:         me.ProcessID,
		ModBaseAddr: uintptr(unsafe.Pointer(me.ModBaseAddr)),
		ModBaseSize: me.ModBaseSize,
		Modules:     map[string]Module{},
	}

	for win.Module32Next(snap, &me) != nil {
		proc.Modules[win.UTF16PtrToString(&me.Module[0])] = Module{
			Name:        win.UTF16PtrToString(&me.Module[0]),
			ModBaseAddr: uintptr(unsafe.Pointer(me.ModBaseAddr)),
			ModBaseSize: me.ModBaseSize,
		}
	}
	win.CloseHandle(snap)
	return proc, nil
}

// Opening processes
func ProcessByPid(pid uint32, bigEndian bool) (Process, error) {
	procInfo, err := processInfo(pid)
	if err != nil {
		return Process{}, err
	}
	err = procInfo.open()
	if err != nil {
		return Process{}, err
	}
	procInfo.BigEndian = bigEndian
	return procInfo, nil
}

func ProcessByName(name string, bigEndian bool) (Process, error) {
	procs := make([]uint32, 0x400)
	var read uint32

	if !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}

	err := win.EnumProcesses(procs, &read)
	if err != nil {
		return Process{}, fmt.Errorf("process %v not found, reason: %v", name, err)
	}

	for _, pid := range procs[:read/4] {
		procInfo, err := processInfo(pid)
		if err != nil {
			continue
			//return Process{}, fmt.Errorf("Could not get processInfo, Reason: %v", err)
		}

		if procInfo.Name == name {
			err = procInfo.open()
			if err != nil {
				return procInfo, err
			}
			procInfo.BigEndian = bigEndian
			return procInfo, nil
		}
	}
	return Process{}, fmt.Errorf("Process '%v' was not found open", name)
}

// Process functions
func (p *Process) open() error {
	handle, err := win.OpenProcess(win.TOKEN_ALL_ACCESS, false, p.Pid)
	if err != nil {
		return fmt.Errorf("can't open process: %v", err.Error())
	}
	p.Handle = handle
	return nil
}

func (p *Process) readBytes(address uintptr, size uintptr) ([]byte, error) {
	data := make([]byte, size)
	var bytesRead uintptr
	err := win.ReadProcessMemory(p.Handle, address, &data[0], size, &bytesRead)
	if err != nil {
		return nil, fmt.Errorf("reading Bytes failed, Reason: %s", err.Error())
	}
	return data, nil
}

// Reading
func (p *Process) ReadInt8(address uintptr) (int8, error) {
	data, err := p.readBytes(address, 1)
	return int8(data[0]), err
}

func (p *Process) ReadInt16(address uintptr) (int16, error) {
	data, err := p.readBytes(address, 2)
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return int16(binary.BigEndian.Uint16(data)), nil
	} else {
		return int16(binary.LittleEndian.Uint16(data)), nil
	}
}

func (p *Process) ReadInt32(address uintptr) (int32, error){
	data, err := p.readBytes(address, 4)
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return int32(binary.BigEndian.Uint32(data)), nil
	} else {
		return int32(binary.LittleEndian.Uint32(data)), nil
	}
}

func (p *Process) ReadInt64(address uintptr) (int64, error){
	data, err := p.readBytes(address, 8
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return int64(binary.BigEndian.Uint64(data)), nil
	} else {
		return int64(binary.LittleEndian.Uint64(data)), nil
	}
}

func (p *Process) ReadFloat32(address uintptr) (float32, error){
	data, err := p.readBytes(address, 4)
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return math.Float32frombits(binary.BigEndian.Uint32(data)), nil
	} else {
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	}
}

func (p *Process) ReadFloat64(address uintptr) (float64, error){
	data, err := p.readBytes(address, 8)
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return math.Float64frombits(binary.BigEndian.Uint64(data)), nil
	} else {
		return math.Float64frombits(binary.LittleEndian.Uint64(data)), nil
	}
}

func (p *Process) ReadSlice(address uintptr, dataType string, size int) (any, error) {
	dataType = strings.ToLower(dataType)
	
	data := make([]any, 3)

	for i := 0; i < size; i++{
		var d any
		var err error

		switch datatype {
			case "int8":
				d, err = p.ReadInt8(address+(i*1))
			case "int16":
				d, err = p.ReadInt16(address+(i*2))
			case "int32":
				d, err = p.ReadInt32(address+(i*4))
			case "int64":
				d, err = p.ReadInt64(address+(i*8))
			case "float32":
				d, err = p.ReadFloat32(address+(i*4))
			case "float64":
				d, err = p.ReadFloat64(address+(i*8))
		}

		if err != nil {
			return 0, err
		}
		data = append(data, d)
	}
	return data, nil
}