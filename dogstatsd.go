package metrics

import (
	"fmt"
	"github.com/Datadog/datadog-go/statsd"
	"strings"
)

// HostnameGetter is a generic function to retrieve a hostname
type HostnameGetter func() string

// DogStatsdSink provides a MetricSink that can be used
// with a dogstatsd server. It utilizes the Dogstatsd client at github.com/Datadog/datadog-go/statsd
type DogStatsdSink struct {
	client                *statsd.Client
	hostnameGetter        HostnameGetter
	enableHostnameTagging bool
}

func getHostname() string {
	conf, _ := GetConfig()
	return conf.HostName
}

// NewDogStatsdSink is used to create a new DogStatsdSink
func NewDogStatsdSink(addr string, tags []string, tagWithHostname bool) (*DogStatsdSink, error) {
	client, err := statsd.New(addr)
	if err != nil {
		return nil, err
	}

	client.Tags = tags
	sink := &DogStatsdSink{
		client:                client,
		hostnameGetter:        getHostname,
		enableHostnameTagging: tagWithHostname,
	}
	return sink, nil
}

func (s *DogStatsdSink) flattenKey(parts []string) string {
	joined := strings.Join(parts, ".")
	return strings.Map(func(r rune) rune {
		switch r {
		case ':':
			fallthrough
		case ' ':
			return '_'
		default:
			return r
		}
	}, joined)
}

func (s *DogStatsdSink) parseKey(key []string) ([]string, []string) {
	var tags []string
	hostName := s.hostnameGetter()

	//Splice the hostname out of the key
	for i, el := range key {
		if el == hostName {
			key = append(key[:i], key[i+1:]...)
		}
	}

	if s.enableHostnameTagging {
		tags = append(tags, fmt.Sprintf("host:%s", hostName))
	}
	return key, tags
}

// Implementation of methods in the MetricSink interface

func (s *DogStatsdSink) SetGauge(key []string, val float32) {
	key, tags := s.parseKey(key)
	flatKey := s.flattenKey(key)

	rate := 1.0
	s.client.Gauge(flatKey, float64(val), tags, rate)
}

func (s *DogStatsdSink) IncrCounter(key []string, val float32) {
	key, tags := s.parseKey(key)
	flatKey := s.flattenKey(key)

	rate := 1.0
	s.client.Count(flatKey, int64(val), tags, rate)
}

// EmitKey is not implemented since DogStatsd does not provide a metric type that holds an
// arbitrary number of values
func (s *DogStatsdSink) EmitKey(key []string, val float32) {
}

func (s *DogStatsdSink) AddSample(key []string, val float32) {
	key, tags := s.parseKey(key)
	flatKey := s.flattenKey(key)

	rate := 1.0
	s.client.TimeInMilliseconds(flatKey, float64(val), tags, rate)
}
