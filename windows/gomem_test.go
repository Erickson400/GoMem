package gomem

import (
	"fmt"
	"testing"
	"time"
)

var playerX uintptr = 0x20AF7700

func TestMem(t *testing.T) {
	proc, err := ProcessByName("pcsx2.exe", false)
	if err != nil {
		t.Error(err)
	}

	for {

		mem, _ := proc.ReadFloat32(playerX + 8)
		proc.WriteFloat32(playerX+8, mem+20)

		fmt.Println(mem)
		time.Sleep(1 * time.Microsecond)
	}
}
