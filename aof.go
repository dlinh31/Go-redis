package main

import (
	"bufio"
	"io"
	"os"
	"sync"
	"time"
)

type Aof struct {
	file *os.File
	rd   *bufio.Reader
	mu   sync.Mutex
}

func NewAof(path string) (*Aof, error) {
	// ✅ Ensure the parent directory exists
	dir := "data" // Path should match what your Go application expects
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// ✅ Create or open the AOF file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	aof := &Aof{
		file: f,
		rd:   bufio.NewReader(f),
	}
	go func() {
		for {
			aof.mu.Lock()
			aof.file.Sync()
			aof.mu.Unlock()
			time.Sleep(time.Second * 1)
		}
	}()
	return aof, nil
}
func (aof *Aof) Close() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	return aof.file.Close()
}

func (aof *Aof) Write(value Value) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	_, err := aof.file.Write(value.Marshal())
	if err != nil {
		return err
	}
	return aof.file.Sync()
}

func (aof *Aof) Read(callback func(value Value)) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	resp := NewResp(aof.file)
	for {
		value, err := resp.Read()
		if err == nil {
			callback(value)
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}

