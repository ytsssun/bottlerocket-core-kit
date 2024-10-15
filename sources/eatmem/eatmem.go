package main

import (
	"fmt"
	"runtime"
	"time"
)

const (
	// Consume 400 MB per iteration
	// 100 MB = 100 * 1024 * 1024 bytes = 104,857,600 bytes
	// Since each int is 4 bytes, we need 104,857,600 / 4 = 26,214,400 ints
	sliceSize = 104_857_600
	
	// Sleep duration between iterations
	sleepDuration = 1 * time.Second
)

func main() {
	var x [][]int
	i := 0

	for {
		i++
		
		// Allocate a new slice of 100 MB and write to it
		newSlice := make([]int, sliceSize)
		for j := range newSlice {
			newSlice[j] = 1  // Write to each element
		}
		x = append(x, newSlice)

		// Get actual memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Calculate memory usage
		allocatedMB := bToMb(m.Alloc)
		totalAllocatedMB := bToMb(m.TotalAlloc)
		systemMB := bToMb(m.Sys)

		fmt.Printf("Iteration: %d, Slices: %d\n", i, len(x))
		fmt.Printf("Allocated: %d MB, Total Allocated: %d MB, System: %d MB\n", 
			allocatedMB, totalAllocatedMB, systemMB)
		fmt.Println("--------------------")

		// Sleep for 2 seconds
		time.Sleep(sleepDuration)
	}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}