package collector

import (
	"runtime"
	"time"
)

// FieldsFunc represents a callback after successfully gathering statistics
type FieldsFunc func(Fields)

// Collector implements the periodic grabbing of informational data from the
// runtime package and outputting the values to a GaugeFunc.
type Collector struct {
	// PauseDur represents the interval in-between each set of stats output.
	// Defaults to 10 seconds.
	PauseDur time.Duration

	// EnableCPU determines whether CPU statistics will be output. Defaults to true.
	EnableCPU bool

	// EnableMem determines whether memory statistics will be output. Defaults to true.
	EnableMem bool

	// EnableGC determines whether garbage collection statistics will be output. EnableMem
	// must also be set to true for this to take affect. Defaults to true.
	EnableGC bool

	// Done, when closed, is used to signal Collector that is should stop collecting
	// statistics and the Run function should return.
	Done <-chan struct{}

	fieldsFunc FieldsFunc
}

// New creates a new Collector that will periodically output statistics to fieldsFunc. It
// will also set the values of the exported fields to the described defaults. The values
// of the exported defaults can be changed at any point before Run is called.
func New(fieldsFunc FieldsFunc) *Collector {
	if fieldsFunc == nil {
		fieldsFunc = func(Fields) {}
	}

	return &Collector{
		PauseDur:   10 * time.Second,
		EnableCPU:  true,
		EnableMem:  true,
		EnableGC:   true,
		fieldsFunc: fieldsFunc,
	}
}

// Run gathers statistics then outputs them to the configured PointFunc every
// PauseDur. Unlike OneOff, this function will return until Done has been closed
// (or never if Done is nil), therefore it should be called in its own go routine.
func (c *Collector) Run() {
	c.fieldsFunc(c.collectStats())

	tick := time.NewTicker(c.PauseDur)
	defer tick.Stop()
	for {
		select {
		case <-c.Done:
			return
		case <-tick.C:
			c.fieldsFunc(c.collectStats())
		}
	}
}

// OneOff gathers returns a map containing all statistics. It is safe for use from
// multiple go routines
func (c *Collector) OneOff() Fields {
	return c.collectStats()
}

func (c *Collector) collectStats() Fields {
	fields := Fields{}

	if c.EnableCPU {
		cStats := cpuStats{
			NumGoroutine: int64(runtime.NumGoroutine()),
			NumCgoCall:   int64(runtime.NumCgoCall()),
			NumCpu:       int64(runtime.NumCPU()),
		}
		c.collectCPUStats(&fields, &cStats)
	}
	if c.EnableMem {
		m := &runtime.MemStats{}
		runtime.ReadMemStats(m)
		c.collectMemStats(&fields, m)
		if c.EnableGC {
			c.collectGCStats(&fields, m)
		}
	}

	fields.Goos = runtime.GOOS
	fields.Goarch = runtime.GOARCH
	fields.Version = runtime.Version()

	return fields
}

func (_ *Collector) collectCPUStats(fields *Fields, s *cpuStats) {
	fields.NumCpu = s.NumCpu
	fields.NumGoroutine = s.NumGoroutine
	fields.NumCgoCall = s.NumCgoCall
}

func (_ *Collector) collectMemStats(fields *Fields, m *runtime.MemStats) {
	// General
	fields.Alloc = int64(m.Alloc)
	fields.TotalAlloc = int64(m.TotalAlloc)
	fields.Sys = int64(m.Sys)
	fields.Lookups = int64(m.Lookups)
	fields.Mallocs = int64(m.Mallocs)
	fields.Frees = int64(m.Frees)

	// Heap
	fields.HeapAlloc = int64(m.HeapAlloc)
	fields.HeapSys = int64(m.HeapSys)
	fields.HeapIdle = int64(m.HeapIdle)
	fields.HeapInuse = int64(m.HeapInuse)
	fields.HeapReleased = int64(m.HeapReleased)
	fields.HeapObjects = int64(m.HeapObjects)

	// Stack
	fields.StackInuse = int64(m.StackInuse)
	fields.StackSys = int64(m.StackSys)
	fields.MSpanInuse = int64(m.MSpanInuse)
	fields.MSpanSys = int64(m.MSpanSys)
	fields.MCacheInuse = int64(m.MCacheInuse)
	fields.MCacheSys = int64(m.MCacheSys)

	fields.OtherSys = int64(m.OtherSys)
}

func (_ *Collector) collectGCStats(fields *Fields, m *runtime.MemStats) {
	fields.GCSys = int64(m.GCSys)
	fields.NextGC = int64(m.NextGC)
	fields.LastGC = int64(m.LastGC)
	fields.PauseTotalNs = int64(m.PauseTotalNs)
	fields.PauseNs = int64(m.PauseNs[(m.NumGC+255)%256])
	fields.NumGC = int64(m.NumGC)
	fields.GCCPUFraction = float64(m.GCCPUFraction)
}

type cpuStats struct {
	NumCpu       int64
	NumGoroutine int64
	NumCgoCall   int64
}

// NOTE: uint64 is not supported by influxDB client due to potential overflows
type Fields struct {
	// CPU
	NumCpu       int64 `json:"cpu.count"`
	NumGoroutine int64 `json:"cpu.goroutines"`
	NumCgoCall   int64 `json:"cpu.cgo_calls"`

	// General
	Alloc      int64 `json:"mem.alloc"`
	TotalAlloc int64 `json:"mem.total"`
	Sys        int64 `json:"mem.sys"`
	Lookups    int64 `json:"mem.lookups"`
	Mallocs    int64 `json:"mem.malloc"`
	Frees      int64 `json:"mem.frees"`

	// Heap
	HeapAlloc    int64 `json:"mem.heap.alloc"`
	HeapSys      int64 `json:"mem.heap.sys"`
	HeapIdle     int64 `json:"mem.heap.idle"`
	HeapInuse    int64 `json:"mem.heap.inuse"`
	HeapReleased int64 `json:"mem.heap.released"`
	HeapObjects  int64 `json:"mem.heap.objects"`

	// Stack
	StackInuse  int64 `json:"mem.stack.inuse"`
	StackSys    int64 `json:"mem.stack.sys"`
	MSpanInuse  int64 `json:"mem.stack.mspan_inuse"`
	MSpanSys    int64 `json:"mem.stack.mspan_sys"`
	MCacheInuse int64 `json:"mem.stack.mcache_inuse"`
	MCacheSys   int64 `json:"mem.stack.mcache_sys"`

	OtherSys int64 `json:"mem.othersys"`

	// GC
	GCSys         int64   `json:"mem.gc.sys"`
	NextGC        int64   `json:"mem.gc.next"`
	LastGC        int64   `json:"mem.gc.last"`
	PauseTotalNs  int64   `json:"mem.gc.pause_total"`
	PauseNs       int64   `json:"mem.gc.pause"`
	NumGC         int64   `json:"mem.gc.count"`
	GCCPUFraction float64 `json:"mem.gc.cpu_fraction"`

	Goarch  string `json:"-"`
	Goos    string `json:"-"`
	Version string `json:"-"`
}

func (f *Fields) Tags() map[string]string {
	return map[string]string{
		"go.os":      f.Goos,
		"go.arch":    f.Goarch,
		"go.version": f.Version,
	}
}

func (f *Fields) Values() map[string]interface{} {
	return map[string]interface{}{
		"cpu.count":      f.NumCpu,
		"cpu.goroutines": f.NumGoroutine,
		"cpu.cgo_calls":  f.NumCgoCall,

		"mem.alloc":   f.Alloc,
		"mem.total":   f.TotalAlloc,
		"mem.sys":     f.Sys,
		"mem.lookups": f.Lookups,
		"mem.malloc":  f.Mallocs,
		"mem.frees":   f.Frees,

		"mem.heap.alloc":    f.HeapAlloc,
		"mem.heap.sys":      f.HeapSys,
		"mem.heap.idle":     f.HeapIdle,
		"mem.heap.inuse":    f.HeapInuse,
		"mem.heap.released": f.HeapReleased,
		"mem.heap.objects":  f.HeapObjects,

		"mem.stack.inuse":        f.StackInuse,
		"mem.stack.sys":          f.StackSys,
		"mem.stack.mspan_inuse":  f.MSpanInuse,
		"mem.stack.mspan_sys":    f.MSpanSys,
		"mem.stack.mcache_inuse": f.MCacheInuse,
		"mem.stack.mcache_sys":   f.MCacheSys,
		"mem.othersys":           f.OtherSys,

		"mem.gc.sys":          f.GCSys,
		"mem.gc.next":         f.NextGC,
		"mem.gc.last":         f.LastGC,
		"mem.gc.pause_total":  f.PauseTotalNs,
		"mem.gc.pause":        f.PauseNs,
		"mem.gc.count":        f.NumGC,
		"mem.gc.cpu_fraction": float64(f.GCCPUFraction),
	}
}
