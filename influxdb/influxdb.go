package influxdb

import (
	"expvar"
	"github.com/tevjef/go-runtime-metrics/collector"
)

// A structure compatible with Telegraf's InfluxDB input plugin format
// https://github.com/influxdata/telegraf/tree/master/plugins/inputs/influxdb
type Point struct {
	Name   string            `json:"name"`
	Tags   map[string]string `json:"tags"`
	Values collector.Fields  `json:"values"`
}

// Metrics returns a expvar.Func which implements Var by calling the function
// and formatting the returned value using JSON. Use this function when you need
// control of the measurement name for a data point.
//
//  package main
//
//  import (
//     "expvar"
//     "github.com/tevjef/go-runtime-metrics/influxdb"
//  )
//
//  func main {
//      expvar.Publish(os.Args[0], influxdb.Metrics("my-measurement-name"))
//  }
//
//
func Metrics(measurement string) expvar.Func {
	c := collector.New(nil)
	return expvar.Func(func() interface{} {
		values := c.OneOff()
		return &Point{
			Name:   measurement,
			Tags:   values.Tags(),
			Values: values,
		}
	})
}
