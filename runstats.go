package runstats

import (
	"log"
	"os"
	"time"

	"fmt"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/tevjef/go-runtime-metrics/collector"
)

const (
	defaultHost               = "localhost:8086"
	defaultMeasurement        = "go.runtime"
	defaultDatabase           = "stats"
	defaultCollectionInterval = 10 * time.Second
	defaultBatchInterval      = 60 * time.Second
)

// A configuration with default values.
var DefaultConfig = &Config{}

type Config struct {
	// InfluxDb host:port pair.
	// Default is "localhost:8086".
	Host string

	// Database to write points to.
	// Default is "stats" and is auto created
	Database string

	// Username with privileges on provided database.
	Username string

	// Password for provided user.
	Password string

	// Measurement to write points to.
	// Default is "go.runtime.<hostname>".
	Measurement string

	// Measurement to write points to.
	RetentionPolicy string

	// Interval at which to write batched points to InfluxDB.
	// Default is 60 seconds
	BatchInterval time.Duration

	// Precision in time to write your points in.
	// Default is nanoseconds
	Precision string

	// Interval at which to collect points.
	// Default is 10 seconds
	CollectionInterval time.Duration

	// Disable collecting CPU Statistics. cpu.*
	// Default is false
	DisableCpu bool

	// Disable collecting Memory Statistics. mem.*
	DisableMem bool

	// Disable collecting GC Statistics (requires Memory be not be disabled). mem.gc.*
	DisableGc bool

	// Default is DefaultLogger which exits when the library encounters a fatal error.
	Logger Logger
}

func (config *Config) init() (*Config, error) {
	if config == nil {
		config = DefaultConfig
	}

	if config.Database == "" {
		config.Database = defaultDatabase
	}

	if config.Host == "" {
		config.Host = defaultHost
	}

	if config.Measurement == "" {
		config.Measurement = defaultMeasurement

		if hn, err := os.Hostname(); err != nil {
			config.Measurement += ".unknown"
		} else {
			config.Measurement += "." + hn
		}
	}

	if config.CollectionInterval == 0 {
		config.CollectionInterval = defaultCollectionInterval
	}

	if config.BatchInterval == 0 {
		config.BatchInterval = defaultBatchInterval
	}

	if config.Logger == nil {
		config.Logger = &DefaultLogger{}
	}

	return config, nil
}

func RunCollector(config *Config) (err error) {
	if config, err = config.init(); err != nil {
		return err
	}

	// Make client
	clnt, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + config.Host,
		Username: config.Username,
		Password: config.Password,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create influxdb client")
	}

	// Ping InfluxDB to ensure there is a connection
	if _, _, err := clnt.Ping(5 * time.Second); err != nil {
		return errors.Wrap(err, "failed to ping influxdb client")
	}

	// Auto create database
	_, err = queryDB(clnt, fmt.Sprintf("CREATE DATABASE \"%s\"", config.Database))

	if err != nil {
		config.Logger.Fatalln(err)
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        config.Database,
		Precision:       config.Precision,
		RetentionPolicy: config.RetentionPolicy,
	})

	if err != nil {
		return err
	}

	_runStats := &runStats{
		logger:      config.Logger,
		client:      clnt,
		points:      bp,
		measurement: config.Measurement,
	}

	go _runStats.loop(config.BatchInterval)

	_collector := collector.New(_runStats.onNewPoint)
	_collector.PauseDur = config.CollectionInterval
	_collector.EnableCPU = !config.DisableCpu
	_collector.EnableMem = !config.DisableMem
	_collector.EnableGC = !config.DisableGc

	go _collector.Run()

	return nil
}

type runStats struct {
	logger      Logger
	client      client.Client
	points      client.BatchPoints
	measurement string
}

func (r *runStats) onNewPoint(fields collector.Fields) {
	pt, err := client.NewPoint(r.measurement, fields.Tags(), fields.Values(), time.Now())
	if err != nil {
		r.logger.Fatalln(errors.Wrap(err, "error while creating point"))
	}

	// log collected points for debugging
	r.logger.Println(pt.String())

	r.points.AddPoint(pt)
}

// Write collected points to influxdb periodically
func (r *runStats) loop(interval time.Duration) {
	for range time.Tick(interval) {
		if len(r.points.Points()) > 0 {
			if err := r.client.Write(r.points); err != nil {
				r.logger.Fatalln(errors.Wrap(err, "error while writing points to influxdb"))
			}
		}
	}
}

type Logger interface {
	Println(v ...interface{})
	Fatalln(v ...interface{})
}

type DefaultLogger struct{}

func (*DefaultLogger) Println(v ...interface{}) {}
func (*DefaultLogger) Fatalln(v ...interface{}) { log.Fatalln(v) }

func queryDB(clnt client.Client, cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command: cmd,
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}
