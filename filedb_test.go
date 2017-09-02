package main

import (
	"testing"
)

func BenchmarkList(b *testing.B) {
	fdb := newFileDatabase("build/feeds")
	var err error
	for i := 0; i < b.N; i++ {
		if err = fdb.list(); err != nil {
			b.Error(err)
		}
	}
}
