package runstats

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/pkg/errors"
	"github.com/tevjef/go-runtime-metrics/collector"
)

const (
	defaultHttpAddr           = "localhost:8086"
	defaultUdpAddr            = "localhost:8089"
	defaultMeasurement        = "go.runtime"
	defaultDatabase           = "stats"
	defaultCollectionInterval = 10 * time.Second
	defaultBatchInterval      = 60 * time.Second
)

// A configuration with default values.
var DefaultConfig = &Config{}

type Config struct {
	// InfluxDb host:port pair.
	// Addr should be of the form "host:port"
	// or "[ipv6-host%zone]:port".
	// Default is "localhost:8086".
	Addr string

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

	UseUDP bool

	// UserAgent is the http User Agent, defaults to "InfluxDBClient".
	UserAgent string

	// Timeout for influxdb writes, defaults to no timeout.
	Timeout time.Duration

	// InsecureSkipVerify gets passed to the http client, if true, it will
	// skip https certificate verification. Defaults to false.
	InsecureSkipVerify bool

	// TLSConfig allows the user to set their own TLS config for the HTTP
	// Client. If set, this option overrides InsecureSkipVerify.
	TLSConfig *tls.Config

	// PayloadSize is the maximum size of a UDP client message, optional
	// Tune this based on your network. Defaults to UDPPayloadSize.
	PayloadSize int
}

func (config *Config) init() (*Config, error) {
	if config == nil {
		config = DefaultConfig
	}

	if config.Database == "" {
		config.Database = defaultDatabase
	}

	if config.Addr == "" {
		if config.UseUDP {
			config.Addr = defaultUdpAddr
		} else {
			config.Addr = defaultHttpAddr
		}
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

	var clnt client.Client

	if config.UseUDP {
		// Make UDP client
		clnt, err = client.NewUDPClient(client.UDPConfig{
			Addr:        config.Addr,
			PayloadSize: config.PayloadSize,
		})

	} else {
		clnt, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:               "http://" + config.Addr,
			Username:           config.Username,
			Password:           config.Password,
			UserAgent:          config.UserAgent,
			Timeout:            config.Timeout,
			InsecureSkipVerify: config.InsecureSkipVerify,
			TLSConfig:          config.TLSConfig,
		})
	}

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

	runStats := &runStats{
		logger: config.Logger,
		client: clnt,
		config: config,
		pc:     make(chan *client.Point),
	}

	bp, err := runStats.newBatch()

	if err != nil {
		return err
	}

	runStats.points = bp

	go runStats.loop(config.BatchInterval)

	c := collector.New(runStats.onNewPoint)
	c.PauseDur = config.CollectionInterval
	c.EnableCPU = !config.DisableCpu
	c.EnableMem = !config.DisableMem
	c.EnableGC = !config.DisableGc

	go c.Run()

	return nil
}

type runStats struct {
	logger Logger
	client client.Client
	points client.BatchPoints
	config *Config
	pc     chan *client.Point
}

func (r *runStats) onNewPoint(fields collector.Fields) {
	pt, err := client.NewPoint(r.config.Measurement, fields.Tags(), fields.Values(), time.Now())

	if err != nil {
		r.logger.Fatalln(errors.Wrap(err, "error while creating point"))
	}

	r.pc <- pt
}

func (r *runStats) newBatch() (bp client.BatchPoints, err error) {
	bp, err = client.NewBatchPoints(client.BatchPointsConfig{
		Database:        r.config.Database,
		Precision:       r.config.Precision,
		RetentionPolicy: r.config.RetentionPolicy,
	})

	if err != nil {
		r.logger.Fatalln(errors.Wrap(err, "could not create BatchPoints"))
	}

	return
}

// Write collected points to influxdb periodically
func (r *runStats) loop(interval time.Duration) {
	ticks := time.Tick(interval)

	for {
		select {
		case <-ticks:
			if r.points == nil || len(r.points.Points()) <= 0 {
				continue
			}

			if err := r.client.Write(r.points); err != nil {
				r.logger.Fatalln(errors.Wrap(err, "could not write points to InfluxDB"))
				continue
			}

			r.points = nil

			bp, err := r.newBatch()

			if err != nil {
				r.logger.Fatalln(errors.Wrap(err, "could not create BatchPoints"))
				continue
			}

			r.points = bp

		case pt := <-r.pc:
			if r.points != nil {
				r.logger.Println(pt.String())

				r.points.AddPoint(pt)
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
