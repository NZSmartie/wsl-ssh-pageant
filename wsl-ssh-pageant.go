package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

const (
	wmCopyData       = 0x004A // WM_COPYDATA
	maxMessageLength = 8192
	copyDataID       = uintptr(0x804e50ba)
)

var (
	// Windows DLLs
	user32   = syscall.NewLazyDLL("User32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	// Win32 API imports
	findWindow         = user32.NewProc("FindWindowW")
	getCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
	createFileMapping  = kernel32.NewProc("CreateFileMapping")
	sendMessage        = user32.NewProc("SendMessageW")

	queryLock sync.Mutex
)

type copyDataStruct struct {
	dwData uintptr
	cbData uint32
	lpData unsafe.Pointer
}

func query(buffer []byte) ([]byte, error) {
	queryLock.Lock()
	defer queryLock.Unlock()

	// fetch the Pageant window.
	hwnd, _, err := findWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Pageant"))))
	if hwnd == 0 {
		return nil, os.NewSyscallError(findWindow.Name, err)
	}

	// var mapName = String.Format("PageantRequest{0:x8}", GetCurrentThreadId());
	threadID, _, _ := getCurrentThreadID.Call()
	mapName := fmt.Sprintf("PageantRequest%08x", threadID)
	pMapName, _ := syscall.UTF16PtrFromString(mapName)

	mmap, err := syscall.CreateFileMapping(syscall.InvalidHandle, nil, syscall.PAGE_READWRITE, 0, maxMessageLength+4, pMapName)
	if err != nil {
		return nil, err
	}

	defer syscall.CloseHandle(mmap)

	ptr, err := syscall.MapViewOfFile(mmap, syscall.FILE_MAP_WRITE, 0, 0, 0)
	if err != nil {
		return nil, err
	}

	defer syscall.UnmapViewOfFile(ptr)

	mmSlice := (*(*[maxMessageLength + 4]byte)(unsafe.Pointer(ptr)))[:]

	// Write our query to the shared memeory
	copy(mmSlice, buffer)

	mapNameBytes := append([]byte(mapName), 0)

	cds := copyDataStruct{
		dwData: copyDataID,
		cbData: uint32(len(mapNameBytes)),
		lpData: unsafe.Pointer(&mapNameBytes[0]),
	}

	// Inform pageant of the share memory file name
	resp, _, err := sendMessage.Call(hwnd, wmCopyData, 0, uintptr(unsafe.Pointer(&cds)))
	if resp == 0 {
		return nil, os.NewSyscallError(sendMessage.Name, errors.New("Pageant was not informed of our query"))
	}

	responseLen := binary.BigEndian.Uint32(mmSlice[:4])

	if responseLen > maxMessageLength {
		return nil, errors.New("Reponse from pagent too large")
	}

	response := make([]byte, responseLen+4)
	copy(response, mmSlice)

	return response, nil
}

func main() {
	inReader := bufio.NewReader(os.Stdin)

	defer os.Stdin.Close()

	for true {
		// Get the 4 byte length from stdin.
		header := make([]byte, 4)
		_, err := inReader.Read(header)
		if err == io.EOF {
			break
		}
		if err != nil {

			log.Fatal(err)
		}
		inputLength := binary.BigEndian.Uint32(header)
		if inputLength > maxMessageLength {
			log.Fatal(errors.New("Request message too large"))
		}

		data := make([]byte, inputLength)
		_, err = inReader.Read(data)
		if err != nil {
			log.Fatal(err)
		}

		data = append(header, data...)

		msg, err := query(data)
		if err != nil {
			log.Fatal(err)
		}

		os.Stdout.Write(msg)
	}
}
