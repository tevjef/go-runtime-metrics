package collector

import (
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test because testing.Short is enabled")
	}

	latestFields := []Fields{}
	pointFunc := func(fields Fields) {
		latestFields = append(latestFields, fields)
	}

	done := make(chan struct{})
	collectorShutdown := make(chan struct{})
	c := New(pointFunc)
	c.PauseDur = 100 * time.Millisecond
	c.Done = done

	go func() {
		defer close(collectorShutdown)
		c.Run()
	}()
	time.Sleep(time.Second)
	close(done)
	<-collectorShutdown

	expKeys := []string{
		"cpu.goroutines",
		"mem.lookups",
		"mem.gc.count",
	}

	for _, fields := range latestFields {
		for _, expKey := range expKeys {
			if _, ok := fields.Values()[expKey]; !ok {
				t.Errorf("expected key (%s) not found", expKey)
			}
		}
	}

	expected := 10
	if points := len(latestFields); points < expected {
		t.Errorf("num of points is lower than expected:\ngot: %d\nexp: %d", points, expected)
	}

}
