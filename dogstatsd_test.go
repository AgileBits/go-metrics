package metrics

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

var EmptyTags []string

const (
	DogStatsdAddr    = "127.0.0.1:7254"
	HostnameEnabled  = true
	HostnameDisabled = false
	TestHostname     = "test_hostname"
)

func MockGetHostname() string {
	return TestHostname
}

var ParseKeyTests = []struct {
	KeyToParse            []string
	Tags                  []string
	EnableHostnameTagging bool
	ExpectedKey           []string
	ExpectedTags          []string
}{
	{[]string{"a", MockGetHostname(), "b", "c"}, EmptyTags, HostnameDisabled, []string{"a", "b", "c"}, EmptyTags},
	{[]string{"a", "b", "c"}, EmptyTags, HostnameDisabled, []string{"a", "b", "c"}, EmptyTags},
	{[]string{"a", "b", "c"}, EmptyTags, HostnameEnabled, []string{"a", "b", "c"}, []string{fmt.Sprintf("host:%s", MockGetHostname())}},
}

var FlattenKeyTests = []struct {
	KeyToFlatten []string
	Expected     string
}{
	{[]string{"a", "b", "c"}, "a.b.c"},
	{[]string{"spaces must", "flatten", "to", "underscores"}, "spaces_must.flatten.to.underscores"},
}

var MetricSinkTests = []struct {
	Method                string
	Metric                []string
	Value                 interface{}
	Tags                  []string
	EnableHostnameTagging bool
	Expected              string
}{
	{"SetGauge", []string{"foo", "bar"}, float32(42), EmptyTags, HostnameDisabled, "foo.bar:42.000000|g"},
	{"SetGauge", []string{"foo", "bar", "baz"}, float32(42), EmptyTags, HostnameDisabled, "foo.bar.baz:42.000000|g"},
	{"AddSample", []string{"sample", "thing"}, float32(4), EmptyTags, HostnameDisabled, "sample.thing:4.000000|ms"},
	{"IncrCounter", []string{"count", "me"}, float32(3), EmptyTags, HostnameDisabled, "count.me:3|c"},

	{"SetGauge", []string{"foo", "baz"}, float32(42), []string{"my_tag:my_value"}, HostnameDisabled, "foo.baz:42.000000|g|#my_tag:my_value"},
	{"SetGauge", []string{"foo", "bar"}, float32(42), []string{"my_tag:my_value", "other_tag:other_value"}, HostnameDisabled, "foo.bar:42.000000|g|#my_tag:my_value,other_tag:other_value"},
	{"SetGauge", []string{"foo", "bar"}, float32(42), []string{"my_tag:my_value", "other_tag:other_value"}, HostnameEnabled, "foo.bar:42.000000|g|#my_tag:my_value,other_tag:other_value,host:test_hostname"},
}

func MockNewDogStatsdSink(addr string, tags []string, tagWithHostname bool) *DogStatsdSink {
	dog, _ := NewDogStatsdSink(addr, tags, tagWithHostname)
	dog.hostnameGetter = MockGetHostname
	return dog
}

func TestParseKey(t *testing.T) {
	for _, tt := range ParseKeyTests {
		dog := MockNewDogStatsdSink(DogStatsdAddr, tt.Tags, tt.EnableHostnameTagging)
		key, tags := dog.parseKey(tt.KeyToParse)

		if !reflect.DeepEqual(key, tt.ExpectedKey) {
			t.Fatalf("Key Parsing failed for %v", tt.KeyToParse)
		}

		if !reflect.DeepEqual(tags, tt.ExpectedTags) {
			t.Fatalf("Tag Parsing Failed for %v", tt.KeyToParse)
		}
	}
}

func TestFlattenKey(t *testing.T) {
	dog := MockNewDogStatsdSink(DogStatsdAddr, EmptyTags, HostnameDisabled)
	for _, tt := range FlattenKeyTests {
		if !reflect.DeepEqual(dog.flattenKey(tt.KeyToFlatten), tt.Expected) {
			t.Fatalf("Flattening %v failed", tt.KeyToFlatten)
		}
	}
}

func TestMetricSink(t *testing.T) {
	udpAddr, err := net.ResolveUDPAddr("udp", DogStatsdAddr)
	if err != nil {
		t.Fatal(err)
	}
	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	buf := make([]byte, 1024)

	for _, tt := range MetricSinkTests {
		dog := MockNewDogStatsdSink(DogStatsdAddr, tt.Tags, tt.EnableHostnameTagging)
		method := reflect.ValueOf(dog).MethodByName(tt.Method)
		method.Call([]reflect.Value{
			reflect.ValueOf(tt.Metric),
			reflect.ValueOf(tt.Value)})

		n, _ := server.Read(buf)
		msg := buf[:n]
		if string(msg) != tt.Expected {
			t.Fatalf("Line %s does not match expected: %s", string(msg), tt.Expected)
		}
	}
}
