package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/hirochachacha/go-smb2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
)

var (
	hostFlag   = flag.String("host", "192.168.1.2", "Air Quality device hostname")
	portFlag   = flag.Int("port", 445, "Air Quality device port")
	userFlag   = flag.String("user", "airvisual", "User name")
	passFlag   = flag.String("pass", "", "Password")
	listenFlag = flag.String("listen", ":1280", "HTTP server address and port")
)

type Measurement struct {
	CO2      float64 `json:"co2_ppm,string"`
	Humidity float64 `json:"humidity_RH,string"`
	PM10     float64 `json:"pm10_ugm3,string"`
	PM25     float64 `json:"pm25_ugm3,string"`
	Temp     float64 `json:"temperature_C,string"`
}

type Measurements struct {
	Measurements []Measurement `json:"measurements"`
}

var (
	CO2Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "co2_ppm",
		Help: "CO2 measurement level in part-per-million (ppm)",
	})
	HumidityGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "humidity",
		Help: "Relative humidity (%)",
	})
	PM10Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pm10",
		Help: "PM10 particles",
	})
	PM25Gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pm25",
		Help: "PM25 particles",
	})
	TempGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "temp",
		Help: "Temperature (C)",
	})
)

func updateMetrics() error {
	conn, err := net.Dial("tcp", *hostFlag+":"+strconv.Itoa(*portFlag))
	if err != nil {
		return err
	}

	defer conn.Close()

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     *userFlag,
			Password: *passFlag,
		},
	}

	s, err := d.Dial(conn)
	if err != nil {
		return err
	}

	defer s.Logoff()

	fs, err := s.Mount("airvisual")

	if err != nil {
		return err
	}

	defer fs.Umount()

	f, err := fs.Open("latest_config_measurements.json")

	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)

	if err != nil {
		return err
	}

	var measurements Measurements

	err = json.Unmarshal(data, &measurements)
	if err != nil {
		return err
	}

	if len(measurements.Measurements) == 0 {
		return errors.New("sensor returned invalid data")
	}

	var measurement *Measurement
	measurement = &measurements.Measurements[0]

	CO2Gauge.Set(measurement.CO2)
	HumidityGauge.Set(measurement.Humidity)
	PM10Gauge.Set(measurement.PM10)
	PM25Gauge.Set(measurement.PM25)
	TempGauge.Set(measurement.Temp)

	return nil
}

var promHandler http.Handler

func MetricHandler(w http.ResponseWriter, r *http.Request) {
	err := updateMetrics()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	promHandler.ServeHTTP(w, r)
}

func main() {
	flag.Parse()

	r := prometheus.NewRegistry()
	r.MustRegister(CO2Gauge, HumidityGauge, PM10Gauge, PM25Gauge, TempGauge)
	promHandler = promhttp.HandlerFor(r, promhttp.HandlerOpts{})

	http.HandleFunc("/metrics", MetricHandler)
	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}
