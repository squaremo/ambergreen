package metcdstore

import "testing"

func TestServiceKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName string
		ok          bool
	}{
		"": {"", false},
		"/weave-flux/service/foo":           {"", false},
		"/weave-flux/service/foo/":          {"", false},
		"/weave-flux/service/foo/details":   {"foo", true},
		"/weave-flux/service/foo/instance":  {"", false},
		"/weave-flux/service/foo/groupspec": {"", false},
	} {
		serviceName, ok := parseServiceKey([]byte(key))
		if serviceName != want.serviceName || ok != want.ok {
			t.Errorf("%q: want (%q %v), have (%q %v)", key, want.serviceName, want.ok, serviceName, ok)
		}
	}
}

func TestInstanceKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName  string
		instanceName string
		ok           bool
	}{
		"": {"", "", false},
		"/weave-flux/service/foo":               {"", "", false},
		"/weave-flux/service/foo/instance/":     {"", "", false},
		"/weave-flux/service/foo/instance/bar":  {"foo", "bar", true},
		"/weave-flux/service/foo/instance/bar/": {"", "", false},
		"/weave-flux/service/foo/groupspec/bar": {"", "", false},
	} {
		serviceName, instanceName, ok := parseInstanceKey([]byte(key))
		if serviceName != want.serviceName || instanceName != want.instanceName || ok != want.ok {
			t.Errorf("%q: want (%q %q %v), have (%q %q %v)", key, want.serviceName, want.instanceName, want.ok, serviceName, instanceName, ok)
		}
	}
}

func TestContainerRuleKeyRegexp(t *testing.T) {
	for key, want := range map[string]struct {
		serviceName       string
		containerRuleName string
		ok                bool
	}{
		"": {"", "", false},
		"/weave-flux/service/foo":                {"", "", false},
		"/weave-flux/service/foo/groupspec/":     {"", "", false},
		"/weave-flux/service/foo/instance/bar":   {"", "", false},
		"/weave-flux/service/foo/groupspec/bar":  {"foo", "bar", true},
		"/weave-flux/service/foo/groupspec/bar/": {"", "", false},
	} {
		serviceName, containerRuleName, ok := parseContainerRuleKey([]byte(key))
		if serviceName != want.serviceName || containerRuleName != want.containerRuleName || ok != want.ok {
			t.Errorf("%q: want (%q %q %v), have (%q %q %v)", key, want.serviceName, want.containerRuleName, want.ok, serviceName, containerRuleName, ok)
		}
	}
}
