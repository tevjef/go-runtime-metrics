package runstats

import (
	"flag"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/tevjef/go-runtime-metrics/collector"
)

var (
	influxDbHost    *string = flag.String("influxdb", "localhost:8086", "host:port pair.")
	database        *string = flag.String("influxdb-database", "", "Database to write points to.")
	username        *string = flag.String("influxdb-username", "", "Username with privileges on provided database.")
	password        *string = flag.String("influxdb-password", "", "Password for provided user.")
	measurement     *string = flag.String("influxdb-measurement", "go.runtime", "Measurement to write points to.")
	retentionPolicy *string = flag.String("influxdb-retention-policy", "", "Retention policy of the points.")

	pause *int  = flag.Int("pause", 10, "Collection pause interval")
	cpu   *bool = flag.Bool("cpu", true, "Collect CPU Statistics")
	mem   *bool = flag.Bool("mem", true, "Collect Memory Statistics")
	gc    *bool = flag.Bool("gc", true, "Collect GC Statistics (requires Memory be enabled)")
)

func init() {
	go runCollector()
}

func runCollector() {
	for !flag.Parsed() {
		// Defer execution of this goroutine.
		runtime.Gosched()

		// Add an initial delay while the program initializes to avoid attempting to collect
		// metrics prior to our flags being available / parsed.
		time.Sleep(1 * time.Second)
	}

	if *database == "" {
		log.Fatalln("error:", "no influxdb database was provided")

	}
	// Make client
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + *influxDbHost,
		Username: *username,
		Password: *password,
	})

	if err != nil {
		log.Fatalln("error:", err)
	}

	if *measurement == "go.runtime" {
		hn, err := os.Hostname()

		if err != nil {
			*measurement += ".unknown"
		} else {
			*measurement += "." + hn
		}
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        *database,
		Precision:       "ns",
		RetentionPolicy: *retentionPolicy,
	})

	if err != nil {
		log.Fatalln("error:", err)
	}

	c := collector.New(func(fields collector.Fields) {
		pt, err := client.NewPoint(*measurement, nil, fields.ToMap(), time.Now())
		if err != nil {
			log.Fatalln("error:", err)
		}
		bp.AddPoint(pt)
	})

	c.PauseDur = time.Duration(*pause) * time.Second
	c.EnableCPU = *cpu
	c.EnableMem = *mem
	c.EnableGC = *gc

	go c.Run()

	for range time.Tick(10 * time.Second) {
		influxClient.Write(bp)
	}
}
