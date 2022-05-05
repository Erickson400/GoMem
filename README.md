# GoMem
#### A Simple Memory Hacking Module for Go.

### Usage:
- This Module Reads & Writes any fundamental Go datatype into an active process.
- It's recommended to be used together with Cheat Engine due to a lack of memory features.
- It's best used as a building ground for a memory hacking engine. You might want to wrap it around some GUI or key inputs.

Check out the [Cheat Sheet](CheatSheet.txt) for all the functions the module provides.
```go
package main

import (
	"fmt"
	gomem "github.com/Erickson400/GoMem"
	"time"
)

/*
	Here is an example of changing a player's X position on a PS2 emulator
*/

var playerX uintptr = 0x20AF7700

func main() {
	// Open the Process as little endian memory
	proc, err := ProcessByName("pcsx2.exe", false)
	if err != nil {
		panic(err)
	}

	// Every microsecond it add 20 to the player's X position
	for {
		mem, _ := proc.ReadFloat32(playerX)
		proc.WriteFloat32(playerX, mem + 20)

		fmt.Println(mem)
		time.Sleep(1 * time.Microsecond)
	}
}

```
### Install for Windows:
Just a normal go module import from github.

In your terminal:
> go get github.com/Erickson400/GoMem/windows

then import it into your go code with 
```go 
import "github.com/Erickson400/GoMem/windows"
```

A Special thanks to the [PyMeow Developer @qb-0](https://github.com/qb-0/PyMeow) for helping me out.
I wanted to make a port of PyMeow to Go, but Instead i made my own twist to his library.


