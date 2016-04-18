package metcdstore

import "fmt"

const (
	root        = "/weave-flux/"
	serviceRoot = root + "service/"
	hostRoot    = root + "host/"
)

func serviceRootKey(serviceName string) []byte {
	return []byte(serviceRoot + serviceName)
}

func serviceKey(serviceName string) []byte {
	return []byte(fmt.Sprintf("%s%s/details", serviceRoot, serviceName))
}

func ruleKey(serviceName, ruleName string) []byte {
	return []byte(fmt.Sprintf("%s%s/groupspec/%s", serviceRoot, serviceName, ruleName))
}

func instanceKey(serviceName, instanceName string) []byte {
	return []byte(fmt.Sprintf("%s%s/instance/%s", serviceRoot, serviceName, instanceName))
}

func hostKey(identity string) []byte {
	return []byte(fmt.Sprintf("%s%s", hostRoot, identity))
}
