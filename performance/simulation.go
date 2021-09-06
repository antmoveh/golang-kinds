/*
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package main

import (
	"log"
	"sync"
	"time"
)

func CallChain() {
	HighCpu()
	HighMemory()
	FrequentGc()
	CreateGoroutine()
	LockContention()
	ChanBlock()
}

const (
	Ki = 1024
	Mi = Ki * Ki
	Gi = Ki * Mi
	Ti = Ki * Gi
	Pi = Ki * Ti
)

// 模拟高cpu占用
func HighCpu() {
	log.Println("high cpu code ..., go pprof profile")
	loop := 10000000000
	for i := 0; i < loop; i++ {
		// do nothing
	}
}

// 模拟高内存占用
func HighMemory() {
	log.Println("high memory code ..., go pprof heap")
	buffer := [][Mi]byte{}
	for len(buffer)*Mi < Gi {
		buffer = append(buffer, [Mi]byte{})
	}
}

// 模拟GC频繁回收
func FrequentGc() {
	log.Println("frequent gc code ..., go pprof allocs")
	_ = make([]byte, 16*Mi)
}

// 模拟创建很多goroutine
func CreateGoroutine() {
	log.Println("create goroutine code ..., go pprof goroutine")
	for i := 0; i < 10; i++ {
		go func() {
			time.Sleep(30 * time.Second)
		}()
	}
}

// 模拟锁争用
func LockContention() {
	log.Println("lock contention code ..., go pprof mutex")
	m := &sync.Mutex{}
	m.Lock()
	go func() {
		time.Sleep(time.Second)
		m.Unlock()
	}()
	m.Lock()
}

// 模拟程序阻塞
func ChanBlock() {
	log.Println("chan block code..., go pprof block")
	<-time.After(time.Second)
}
