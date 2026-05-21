package tools

import "log"

// SafeGo runs fn in a goroutine and recovers panics.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
			}
		}()
		fn()
	}()
}
