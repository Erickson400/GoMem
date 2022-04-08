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
		fmt.Println(err)
	}

	for {
		mem, _ := proc.ReadSlice(playerX, "int8", 5)

		for _, v := range mem.([]interface{}) {
			fmt.Print(v)
		}
		fmt.Println("")
		time.Sleep(2 * time.Millisecond)
	}

}
