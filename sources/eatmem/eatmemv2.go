package main

import (
    "fmt"
    "math/rand"
    "os"
    "os/exec"
    "os/signal"
    "runtime"
    "syscall"
    "time"
)

func allocateAndAccessMemory(sizeMB int, accessInterval time.Duration) {
    sizeBytes := sizeMB * 1024 * 1024
    memory := make([]byte, sizeBytes)

    for {
        // Write to random locations
        for i := 0; i < 1000; i++ {
            offset := rand.Intn(sizeBytes)
            memory[offset] = byte(rand.Intn(256))
        }

        time.Sleep(accessInterval)
    }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    var si syscall.Sysinfo_t
    err := syscall.Sysinfo(&si)
    if err != nil {
        fmt.Println("Error getting system info:", err)
        return
    }

    totalMemory := si.Totalram * uint64(si.Unit)
    availableMemoryMB := int(totalMemory / (1024 * 1024))

    // Allocate 90% of available memory across 5 processes
    memoryPerProcess := int(float64(availableMemoryMB) * 0.7 / 5)

    fmt.Printf("Total memory: %d MB\n", availableMemoryMB)
    fmt.Printf("Memory per process: %d MB\n", memoryPerProcess)

    processes := make([]*os.Process, 5)

    for i := 0; i < 5; i++ {
        interval := i + 1 // intervals from 1 to 5 seconds in milliseconds
        cmd := exec.Command(os.Args[0], "child", fmt.Sprintf("%d", memoryPerProcess), fmt.Sprintf("%d", interval))
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        err := cmd.Start()
        if err != nil {
            fmt.Printf("Error starting process %d: %v\n", i, err)
            continue
        }

        processes[i] = cmd.Process
        fmt.Printf("Started process %d with PID %d, interval %dms\n", i, cmd.Process.Pid, interval)
        time.Sleep(1 * time.Second)
    }

    if len(os.Args) > 1 && os.Args[1] == "child" {
        sizeMB := 0
        interval := 0
        fmt.Sscanf(os.Args[2], "%d", &sizeMB)
        fmt.Sscanf(os.Args[3], "%d", &interval)
        allocateAndAccessMemory(sizeMB, time.Duration(interval)*time.Second)
        return
    }

    // Handle Ctrl+C
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)

    <-c // Wait for Ctrl+C

    fmt.Println("Terminating child processes...")
    for _, process := range processes {
        if process != nil {
            process.Kill()
        }
    }
}