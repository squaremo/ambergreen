package metcdstore

import "fmt"

const (
	root        = "/weave-flux/"
	serviceRoot = root + "service/"
	hostRoot    = root + "host/"
)

func serviceRootKey(serviceName string) string {
	return serviceRoot + serviceName
}

func serviceKey(serviceName string) string {
	return fmt.Sprintf("%s%s/details", serviceRoot, serviceName)
}

func ruleKey(serviceName, ruleName string) string {
	return fmt.Sprintf("%s%s/groupspec/%s", serviceRoot, serviceName, ruleName)
}

func instanceKey(serviceName, instanceName string) string {
	return fmt.Sprintf("%s%s/instance/%s", serviceRoot, serviceName, instanceName)
}
