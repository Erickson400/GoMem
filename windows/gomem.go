package gomem

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unsafe"

	win "golang.org/x/sys/windows"
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
		ModBaseAddr: me.ModBaseAddr,
		ModBaseSize: me.ModBaseSize,
		Modules:     map[string]Module{},
	}

	for win.Module32Next(snap, &me) != nil {
		proc.Modules[win.UTF16PtrToString(&me.Module[0])] = Module{
			Name:        win.UTF16PtrToString(&me.Module[0]),
			ModBaseAddr: me.ModBaseAddr,
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

func (p *Process) ReadInt32(address uintptr) (int32, error) {
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

func (p *Process) ReadInt64(address uintptr) (int64, error) {
	data, err := p.readBytes(address, 8)
	if err != nil {
		return 0, err
	}
	if p.BigEndian {
		return int64(binary.BigEndian.Uint64(data)), nil
	} else {
		return int64(binary.LittleEndian.Uint64(data)), nil
	}
}

func (p *Process) ReadFloat32(address uintptr) (float32, error) {
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

func (p *Process) ReadFloat64(address uintptr) (float64, error) {
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

	for i := uintptr(0); i < uintptr(size); i++ {
		var d any
		var err error

		switch dataType {
		case "int8":
			d, err = p.ReadInt8(address + (i * 1))
		case "int16":
			d, err = p.ReadInt16(address + (i * 2))
		case "int32":
			d, err = p.ReadInt32(address + (i * 4))
		case "int64":
			d, err = p.ReadInt64(address + (i * 8))
		case "float32":
			d, err = p.ReadFloat32(address + (i * 4))
		case "float64":
			d, err = p.ReadFloat64(address + (i * 8))
		}

		if err != nil {
			return 0, err
		}
		data = append(data, d)
	}
	return data, nil
}

// Writing
func (p *Process) writeBytes(address uintptr, data []byte) error {
	var written uintptr
	err := win.WriteProcessMemory(p.Handle, address, &data[0], uintptr(len(data)), &written)
	if err != nil {
		return fmt.Errorf("ERROR: cant write bytes, Reason: %v", err)
	}
	return nil
}

func (p *Process) WriteInt8(address uintptr, data int8) error {
	d := make([]byte, 1)
	d[0] = byte(data)
	err := p.writeBytes(address, d)
	if err != nil {
		return err
	}
	return nil
}

func (p *Process) WriteInt16(address uintptr, data int16) error {
	d := make([]byte, 2)
	if p.BigEndian {
		binary.BigEndian.PutUint16(d, uint16(data))
	} else {
		binary.LittleEndian.PutUint16(d, uint16(data))
	}
	p.writeBytes(address, d)
	return nil
}

func (p *Process) WriteInt32(address uintptr, data int32) error {
	d := make([]byte, 4)
	if p.BigEndian {
		binary.BigEndian.PutUint32(d, uint32(data))
	} else {
		binary.LittleEndian.PutUint32(d, uint32(data))
	}
	p.writeBytes(address, d)
	return nil
}

func (p *Process) WriteInt64(address uintptr, data int64) error {
	d := make([]byte, 8)
	if p.BigEndian {
		binary.BigEndian.PutUint64(d, uint64(data))
	} else {
		binary.LittleEndian.PutUint64(d, uint64(data))
	}
	p.writeBytes(address, d)
	return nil
}

func (p *Process) WriteFloat32(address uintptr, data float32) error {
	d := make([]byte, 4)
	if p.BigEndian {
		binary.BigEndian.PutUint32(d, math.Float32bits(data))
	} else {
		binary.LittleEndian.PutUint32(d, math.Float32bits(data))
	}
	p.writeBytes(address, d)
	return nil
}

func (p *Process) WriteFloat64(address uintptr, data float64) error {
	d := make([]byte, 8)
	if p.BigEndian {
		binary.BigEndian.PutUint64(d, math.Float64bits(data))
	} else {
		binary.LittleEndian.PutUint64(d, math.Float64bits(data))
	}
	p.writeBytes(address, d)
	return nil
}

type Vector3 struct {
	x, y, z float32
}

func (p *Process) ReadVector3(address uintptr) Vector3 {
	var err error
	vec := Vector3{0, 0, 0}
	vec.x, err = p.ReadFloat32(address)
	vec.y, err = p.ReadFloat32(address + 4)
	vec.z, err = p.ReadFloat32(address + 8)
	if err != nil {
		fmt.Printf("Could not read vector 3 %s", err)
		return Vector3{0, 0, 0}
	}
	return vec
}

func (p *Process) WriteVector3(address uintptr, vec Vector3) {
	var err error
	err = p.WriteFloat32(address, vec.x)
	err = p.WriteFloat32(address+4, vec.y)
	err = p.WriteFloat32(address+8, vec.z)
	if err != nil {
		fmt.Printf("Could not write vector 3 %s", err)
	}
}

func (v *Vector3) Add(vec Vector3) {
	v.x += vec.x
	v.y += vec.y
	v.z += vec.z
}

func (v *Vector3) Sub(vec Vector3) {
	v.x -= vec.x
	v.y -= vec.y
	v.z -= vec.z
}

func (v *Vector3) Mul(vec Vector3) {
	v.x += vec.x
	v.y += vec.y
	v.z += vec.z
}

func (v *Vector3) Div(vec Vector3) {
	v.x /= vec.x
	v.y /= vec.y
	v.z /= vec.z
}

type Vector2 struct {
	x, y float32
}

func (p *Process) ReadVector2(address uintptr) Vector2 {
	var err error
	vec := Vector2{0, 0}
	vec.x, err = p.ReadFloat32(address)
	vec.y, err = p.ReadFloat32(address + 4)
	if err != nil {
		fmt.Printf("Could not read vector 2 %s", err)
		return Vector2{0, 0}
	}
	return vec
}

func (p *Process) WriteVector2(address uintptr, vec Vector2) {
	var err error
	err = p.WriteFloat32(address, vec.x)
	err = p.WriteFloat32(address+4, vec.y)
	if err != nil {
		fmt.Printf("Could not write vector 2 %s", err)
	}
}

func (v *Vector2) Add(vec Vector2) {
	v.x += vec.x
	v.y += vec.y
}

func (v *Vector2) Sub(vec Vector2) {
	v.x -= vec.x
	v.y -= vec.y
}

func (v *Vector2) Mul(vec Vector2) {
	v.x += vec.x
	v.y += vec.y
}

func (v *Vector2) Div(vec Vector2) {
	v.x /= vec.x
	v.y /= vec.y
}
