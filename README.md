# go-runtime-metrics
Collect Golang Runtime Metrics, outputting to a influxdb. Inspired by https://github.com/bmhatfield/go-runtime-metrics

The intent of this library is to be a "side effect" import. You can kick off the collector merely by importing this into your main:

`import _ "github.com/tevjef/go-runtime-metrics"`

This library has a few optional flags it depends on and one required flag `-influxdb-database`. It won't be able to output stats until you call flag.Parse(), 
which is generally done in your `func main() {}`.

Once imported and running, you can expect a number of Go runtime metrics to be sent to influxdb. 
An example of what this looks like when configured to work with [Grafana](http://grafana.org/):

![](/grafana.png)

```
	-cpu=true 		                collect CPU statistics
	-mem=true			            collect memory statistics
	-gc=true 			            collect GC statistics (requires memory be enabled)
	-pause=10 		                collection pause interval
	-influxdb=localhost:8086        host:port pair.
	-influxdb-database=REQUIRED 	database to write points to.
	-influxdb-username="" 		    username with privileges on provided database.
	-influxdb-password="" 		    password for provided user.
	-influxdb-measurement="" 	    measurement to write points to..
	-influxdb-retention-policy="" 	retention policy of the points.
```
### expvar

* Metric names are easily parsed by regexp.
* Lighter than the standard library memstat expvar
* Includes stats for `cpu.cgo_calls`, `cpu.goroutines` and timing of the last GC pause with `mem.gc.pause`.
* Works out the box with Telegraf's [InfluxDB input plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/influxdb)

Import the expvar package with `import _ "github.com/tevjef/go-runtime-metrics/expvar"` to export metrics with default configurations.
```json
{
  "/go/bin/binary": {
    "name": "go_runtime_metrics",
    "tags": null,
    "values": {
      "cpu.cgo_calls": 1,
      "cpu.goroutines": 2,
      "mem.alloc": 667576,
      "mem.frees": 104,
      "mem.gc.count": 0,
      "mem.gc.last": 0,
      "mem.gc.next": 4194304,
      "mem.gc.pause": 0,
      "mem.gc.pause_total": 0,
      "mem.gc.sys": 65536,
      "mem.heap.alloc": 667576,
      "mem.heap.idle": 475136,
      "mem.heap.inuse": 1327104,
      "mem.heap.objects": 5227,
      "mem.heap.released": 0,
      "mem.heap.sys": 1802240,
      "mem.lookups": 3,
      "mem.malloc": 5331,
      "mem.othersys": 820558,
      "mem.stack.inuse": 294912,
      "mem.stack.mcache_inuse": 4800,
      "mem.stack.mcache_sys": 16384,
      "mem.stack.mspan_inuse": 14160,
      "mem.stack.mspan_sys": 16384,
      "mem.stack.sys": 294912,
      "mem.sys": 3018752,
      "mem.total": 667576
    }
  }
}
```

#### Custom measurement name

`influxdb.Metrics` returns a `expvar.Func` which implements `Var` by calling the function
and formatting the returned value using JSON. Use this function when you need control of the measurement name for a
data point.

```go
package main

import (
   "expvar"
   "github.com/tevjef/go-runtime-metrics/influxdb"
)

func main {
    expvar.Publish(os.Args[0], influxdb.Metrics("my-measurement-name"))
}
```

#### Benchmark

Benchmark against standard library memstat expvar: 
```
$ go test -bench=. -parallel 16 -cpu 1,2,4

BenchmarkMetrics          200000             10838 ns/op            3827 B/op         10 allocs/op
BenchmarkMetrics-2         20000             68876 ns/op            3886 B/op         10 allocs/op
BenchmarkMetrics-4         50000             22368 ns/op            3882 B/op         10 allocs/op
BenchmarkMemstat           20000             78009 ns/op           52264 B/op         12 allocs/op
BenchmarkMemstat-2         10000            155930 ns/op           52264 B/op         12 allocs/op
BenchmarkMemstat-4         10000            144849 ns/op           52266 B/op         12 allocs/op

```


```
  System Info: 

  Processor Name:	Intel Core i5
  Processor Speed:	3.5 GHz
  Number of Processors:	1
  Total Number of Cores:	4
  L2 Cache (per Core):	256 KB
  L3 Cache:	6 MB
  Memory:	32 GB
  Bus Speed:	400 MHz

```
