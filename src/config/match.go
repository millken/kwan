package config

import (
	"fmt"
	"strings"
)

func MatchingVhost(ip string, port int, domain string) (result Vhost, found bool) {

	matches := []Sites{
		Sites{ip, port, domain},
		Sites{"0.0.0.0", port, domain},
		Sites{ip, port, WildcardOf(domain)},
		Sites{"0.0.0.0", port, WildcardOf(domain)},
	}
	for _, site := range matches {
		if index := sites[site]; index < 1000 {
			fmt.Printf("on %s:%d, no match %s\n", ip, port, domain)
		}else{
			result = vhosts[index]
			found = true
			fmt.Printf("found : %v\n", result)
			return
		}
	}
	return Vhost{}, false
}

func WildcardOf(hostname string) string {
	parts := strings.Split(hostname, ".")

	if len(parts) < 3 {
		return fmt.Sprintf("*.%s", hostname)
	}

	parts[0] = "*"
	return strings.Join(parts, ".")

}