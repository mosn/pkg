package utils

import (
	"testing"
	"time"
)

func TestPooledTimer(t *testing.T) {
	start := time.Now()
	t1 := AcquireTimer(time.Second)
	<-t1.C
	// should less than 1/100s
	if time.Since(start).Seconds()-1 > 0.01 {
		t.Fail()
	}

	ReleaseTimer(t1)

	start2 := time.Now()
	t2 := AcquireTimer(time.Second * 2)
	<-t2.C
	// should less than 1/100s
	if time.Since(start2).Seconds()-2 > 0.01 {
		t.Fail()
	}

	ReleaseTimer(t2)
}

func BenchmarkPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tm := AcquireTimer(time.Microsecond)
		<-tm.C
		ReleaseTimer(tm)
	}
	b.StopTimer()
	b.ReportAllocs()
}

func BenchmarkStd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tm := time.NewTimer(time.Microsecond)
		<-tm.C
	}
	b.StopTimer()
	b.ReportAllocs()
}
