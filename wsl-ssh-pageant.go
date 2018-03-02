package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/hidez8891/shm"
)

const (
	WM_COPYDATA = 0x004A
)

const AgentMaxMessageLength int32 = 8192

const AGENT_MAX_MSGLEN int32 = 8192

const AGENT_COPYDATA_ID uintptr = uintptr(0x804e50ba)

var (
	user32             = syscall.NewLazyDLL("User32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	FindWindow         = user32.NewProc("FindWindowW")
	GetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
	SendMessage        = user32.NewProc("SendMessage")
)

type COPYDATASTRUCT struct {
	dwData uintptr // Any value the sender chooses.  Perhaps its main window handle?
	cbData int32   // The count of bytes in the message.
	lpData uintptr // The address of the message.
}

func query(buffer []byte) ([]byte, error) {
	hwnd, _, err := FindWindow.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Pageant"))))
	if hwnd == 0 {
		log.Fatal(os.NewSyscallError(FindWindow.Name, err))
	}

	// var mapName = String.Format("PageantRequest{0:x8}", GetCurrentThreadId());
	threadID, _, _ := GetCurrentThreadId.Call()
	mapName := fmt.Sprintf("PageantRequest%8x", threadID)

	if err != nil {
		return nil, err
	}

	// var fileMap = CreateFileMapping(INVALID_HANDLE_VALUE, IntPtr.Zero, FileMapProtection.PageReadWrite, 0, AGENT_MAX_MSGLEN, mapName);
	sharedMemory, err := shm.Create(mapName, AgentMaxMessageLength)
	if err != nil {
		return nil, err
	}
	defer sharedMemory.Close()

	// Marshal.Copy(buf, 0, sharedMemory, buf.Length);
	_, err = sharedMemory.Write(buffer)
	if err != nil {
		return nil, err
	}

	cds := COPYDATASTRUCT{}
	cds.dwData = AGENT_COPYDATA_ID
	cds.cbData = int32(len(mapName) + 1)

	// bar[bar.Length - 1] = 0;
	// var gch = GCHandle.Alloc(bar);

	// cds.lpData = Marshal.UnsafeAddrOfPinnedArrayElement(bar, 0);
	mapNameBytes := append([]byte(mapName), []byte{0}...)
	copySlice2Ptr(mapNameBytes, cds.lpData, 0, int32(len(mapNameBytes)))

	// var data = Marshal.AllocHGlobal(Marshal.SizeOf(cds));
	// Marshal.StructureToPtr(cds, data, false);
	data := new(bytes.Buffer)
	err = binary.Write(data, binary.LittleEndian, &cds)
	var dataPtr uintptr
	copySlice2Ptr(data.Bytes(), dataPtr, 0, int32(data.Len()))

	// var rcode = SendMessage(hwnd, WM_COPYDATA, IntPtr.Zero, data);
	_, _, err = SendMessage.Call(hwnd, WM_COPYDATA, 0, dataPtr)
	if err != nil {
		return nil, os.NewSyscallError(SendMessage.Name, err)
	}

	len := make([]byte, 4)
	sharedMemory.Read(len)
	// var len = (Marshal.ReadByte(sharedMemory, 0) << 24) |
	// 			(Marshal.ReadByte(sharedMemory, 1) << 16) |
	// 			(Marshal.ReadByte(sharedMemory, 2) << 8) |
	// 			(Marshal.ReadByte(sharedMemory, 3));

	// var ret = new byte[len + 4];
	// Marshal.Copy(sharedMemory, ret, 0, len + 4);
	len = make([]byte, binary.BigEndian.Uint32(len)+4)
	sharedMemory.Read(len)

	return len, nil
}

func main() {

	// Buffer for reading data
	inBuffer := new(bytes.Buffer)

	defer os.Stdin.Close()
	// Loop to receive all the data sent by the client.
	for true {
		inBuffer.Reset()
		i, err := io.Copy(inBuffer, os.Stdin)
		if i == 0 || err != nil {
			break
		}

		msg, err := query(inBuffer.Bytes())
		if err != nil {
			log.Fatal(err)
		}

		outReader := bytes.NewReader(msg)
		io.Copy(os.Stdout, outReader)
	}
}
