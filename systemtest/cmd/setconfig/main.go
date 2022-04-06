package main

import (
	"flag"
	"log"
	"os"

	"github.com/elastic/apm-server/systemtest/fleettest"
)

var (
	kibana     = flag.String("kibana", "http://localhost:5601", "Kibana URL")
	policyName = flag.String("policy", "", "Fleet integration name")
)

func main() {
	flag.Parse()
	if *policyName == "" {
		log.Println("specify the Fleet integration policy name to modify with -policy")
		os.Exit(2)
	}

	client := fleettest.NewClient(*kibana)
	policies, err := client.PackagePolicies("ingest-package-policies.name:" + *policyName)
	if err != nil {
		log.Fatal(err)
	}
	if len(policies) != 1 {
		log.Fatalf("expected 1 matching policy, got %d", len(policies))
	}
	policy := &policies[0]

	// TODO(axw) create flags or a DSL for updating config, vars, etc.
	config := policy.Inputs[0].Config
	config["apm-server.pprof.enabled"] = map[string]interface{}{"value": true}

	if err := client.UpdatePackagePolicy(policy); err != nil {
		log.Fatalf("updating package policy failed: %s", err)
	}
	log.Println("package policy updated")
}
