// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package systemtest

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Kubernetes *kubernetes.Clientset
)

func TestMain(m *testing.M) {
	// Wait up to 5 minutes for the stack to be ready. The system
	// tests assume that "tilt up" is run first, but not that they
	// are entirely ready.
	log.Println("INFO: waiting for stack to be ready...")
	if err := waitTilt(); err != nil {
		log.Fatal(err)
	}

	log.Println("INFO: cleaning up Elasticsearch...")
	if err := cleanupElasticsearch(); err != nil {
		log.Fatal(err)
	}

	// The local integration package will have been built, uploaded,
	// and installed.
	cmd := exec.Command("make", "--no-print-directory", "get-version")
	cmd.Dir = filepath.Join(systemtestDir, "..")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	packageVersion := strings.TrimSpace(string(output))
	IntegrationPackage, err = Fleet.Package("apm", packageVersion)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("INFO: running system tests...")
	os.Exit(m.Run())
}

func waitTilt() error {
	// Wait for all Tilt resources to be up to date and ready.
	for _, condition := range []string{"UpToDate", "Ready"} {
		cmd := exec.Command(
			"tilt", "wait", "--for=condition="+condition,
			"--timeout", "5m",
			"--all", "uiresource",
		)
		cmd.Dir = repoRootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Get the kubeconfig location from Tilt, and initialise a Kubernetes clientset.
	var buf bytes.Buffer
	cmd := exec.Command("tilt", "get", "-o", "json", "cluster/default")
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	configPath := gjson.GetBytes(buf.Bytes(), "status.connection.kubernetes.configPath")
	if !configPath.Exists() {
		io.Copy(os.Stderr, &buf)
		return errors.New("failed to locate kubeconfig path")
	}
	config, err := clientcmd.BuildConfigFromFlags("", configPath.String())
	if err != nil {
		return err
	}
	Kubernetes, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return nil
}
