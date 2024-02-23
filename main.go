package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
	MAX_PATH                  = 260
)

var (
	moduser32               = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = moduser32.NewProc("GetForegroundWindow")
	procGetKeyState         = moduser32.NewProc("GetKeyState")
	modkernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleFileName   = modkernel32.NewProc("GetModuleFileNameW")
)

// type getWindowThreadProcessIdFunc func(hwnd syscall.Handle, lpdwProcessId *uint32) (threadId uintptr, processId uintptr)
func main() {
	// Initialize channel for catching SIGINT signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Obtain a pointer to GetWindowThreadProcessId
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetProcAddress := kernel32.NewProc("GetProcAddress")
	user32 := syscall.NewLazyDLL("user32.dll")
	modUser32 := user32.Handle()
	var ptr uintptr
	// Define the function signature for GetWindowThreadProcessId
	getWinThreadProcId := func(hwnd syscall.Handle, lpdwProcessId *uint32) uintptr {
		ret, _, _ := syscall.Syscall(ptr, 2, uintptr(hwnd), uintptr(unsafe.Pointer(lpdwProcessId)), 0)
		return ret
	}
	// Obtain the function pointer
	ptr, _, _ = procGetProcAddress.Call(uintptr(modUser32), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("GetWindowThreadProcessId"))))
	// Convert the function pointer to a callback
	getWindowThreadProcessId := syscall.NewCallback(getWinThreadProcId)

	// Infinite loop to capture keyboard events
	for {
		select {
		case <-signalChan:
			// Close the program on SIGINT signal
			fmt.Println("Exiting...")
			return
		default:
			// Check if CTRL + SHIFT + . is pressed
			if isCtrlShiftDotPressed() {
				activeWindow := getForegroundWindow()
				if activeWindow != 0 {
					// Get the process ID of the active window
					var processID uint32
					_, _, _ = syscall.Syscall(getWindowThreadProcessId, 2, uintptr(activeWindow), uintptr(unsafe.Pointer(&processID)), 0)
					processHandle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, uint32(processID))

					if err != nil {
						fmt.Println("Error:", err)
						return
					}
					defer syscall.CloseHandle(processHandle)

					// Get the executable path of the process
					executablePath, err := getModuleFileName(processHandle)
					if err != nil {
						fmt.Println("Error:", err)
						return
					}

					// Check if the active window belongs to File Explorer
					if executablePath == "explorer.exe" {
						// Get the path of the current folder in File Explorer
						path, err := getActiveExplorerPath()
						if err != nil {
							fmt.Println("Error:", err)
							return
						}
						fmt.Println("Path:", path)
						// Show your modal here
					}
				}
			}
		}
	}
}

func isCtrlShiftDotPressed() bool {
	// Check if CTRL + SHIFT + . is pressed
	return (GetKeyState(0x11) < 0 && GetKeyState(0x10) < 0 && GetKeyState(0xBE) < 0)
}

func getForegroundWindow() syscall.Handle {
	// Get the handle of the foreground window
	ret, _, _ := procGetForegroundWindow.Call()
	return syscall.Handle(ret)
}

func getModuleFileName(handle syscall.Handle) (string, error) {
	// Get the module file name of the given process handle
	var buffer [MAX_PATH]uint16
	ret, _, err := procGetModuleFileName.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if ret == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buffer[:]), nil
}

func getActiveExplorerPath() (string, error) {
	// TODO: Implement getting the path of the active explorer window
	return "", nil
}

func GetKeyState(nVirtKey int32) int16 {
	// Get the state of the specified virtual key
	ret, _, _ := syscall.Syscall(procGetKeyState.Addr(), 1, uintptr(nVirtKey), 0, 0)
	return int16(ret)
}
