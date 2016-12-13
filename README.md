# go-runtime-metrics
Collect golang runtime metrics, pushing to [InfluxDB](https://www.influxdata.com/time-series-platform/influxdb/) or pulling with [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/). Inspired by https://github.com/bmhatfield/go-runtime-metrics

## Installation

    go get -u github.com/tevjef/go-runtime-metrics
    
## Push Usage

This library can be configured to push metrics directly to InfluxDB.

```go
import (
	metrics "github.com/tevjef/go-runtime-metrics"
)

func main() {
	err := metrics.RunCollector(metrics.DefaultConfig)
	
	if err != nil {
	   // handle error
	}
}
	
```

Once imported and running, you can expect a number of Go runtime metrics to be sent to InfluxDB. 
An example of what this looks like when configured to work with [Grafana](http://grafana.org/):

![](/grafana.png)

[Download Dashboard](https://grafana.net/dashboards/1144)

## Pull Usage via [expvar](https://golang.org/pkg/expvar/)

Package [expvar](https://golang.org/pkg/expvar/) provides a standardized interface to public variables. This library provides an exported InfluxDB formatted variable with a few other benefits: 

* Metric names are easily parsed by regexp.
* Lighter than the standard library memstat expvar
* Includes stats for `cpu.cgo_calls`, `cpu.goroutines` and timing of the last GC pause with `mem.gc.pause`.
* Works out the box with Telegraf's [InfluxDB input plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/influxdb)

Import this library's expvar package with `import _ "github.com/tevjef/go-runtime-metrics/expvar"` to export a variable with default configurations.
```json
{
  "/go/bin/binary": {
    "name": "go_runtime_metrics",
    "tags": {
      "go.arch": "amd64",
      "go.os": "darwin",
      "go.version": "go1.7.4"
    },
    "values": {
      "cpu.count": 4,
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

#### Configuring with [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/)

Your program must import `_ "github.com/tevjef/go-runtime-metrics/expvar` in order for an InfluxDB formatted variable to be exported via `/debug/vars`.

1. [Install Telegraf](https://github.com/influxdata/telegraf#installation)

2. Make a config file utilizing the influxdb input plugin and an output plugin of your choice.

    ```toml
    [[inputs.influxdb]]
      urls = ["http://localhost:6060/debug/vars"]
    
    [[outputs.influxdb]]
      urls = ["http://localhost:8086"]
      ## The target database for metrics (telegraf will create it if not exists).
      database = "stats" # required
      
    ## [[outputs.file]]
    ##   files = ["stdout"]
    ##   data_format = "json"
    ```

3. Start the Telegraf agent with `telegraf -config config.conf`


#### Benchmarks

Benchmark against standard library memstat expvar: 
```
$ go test -bench=. -parallel 16 -cpu 1,2,4

BenchmarkMetrics          100000             12456 ns/op            4226 B/op         21 allocs/op
BenchmarkMetrics-2         20000             63597 ns/op            4264 B/op         21 allocs/op
BenchmarkMetrics-4         50000             28797 ns/op            4266 B/op         21 allocs/op
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
