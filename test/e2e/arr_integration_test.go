//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/poiley/nebularr-operator/test/e2e/containers"
	"github.com/poiley/nebularr-operator/test/utils"
)

// These tests verify the operator works correctly against real *arr containers.
// They require Docker to be running and will spin up actual containers.
var _ = Describe("Arr Integration", Ordered, Label("integration"), func() {
	var (
		ctx               context.Context
		cancel            context.CancelFunc
		radarrContainer   *containers.ArrContainer
		sonarrContainer   *containers.ArrContainer
		prowlarrContainer *containers.ArrContainer
	)

	BeforeAll(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)

		By("starting Radarr container")
		var err error
		radarrContainer, err = containers.StartRadarr(ctx, containers.ArrContainerOptions{
			StartupTimeout: 3 * time.Minute,
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to start Radarr container")
		GinkgoWriter.Printf("Radarr started at %s with API key: %s\n", radarrContainer.URL(), radarrContainer.APIKey)

		By("starting Sonarr container")
		sonarrContainer, err = containers.StartSonarr(ctx, containers.ArrContainerOptions{
			StartupTimeout: 3 * time.Minute,
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to start Sonarr container")
		GinkgoWriter.Printf("Sonarr started at %s with API key: %s\n", sonarrContainer.URL(), sonarrContainer.APIKey)

		By("starting Prowlarr container")
		prowlarrContainer, err = containers.StartProwlarr(ctx, containers.ArrContainerOptions{
			StartupTimeout: 3 * time.Minute,
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to start Prowlarr container")
		GinkgoWriter.Printf("Prowlarr started at %s with API key: %s\n", prowlarrContainer.URL(), prowlarrContainer.APIKey)
	})

	AfterAll(func() {
		By("cleaning up containers")
		if radarrContainer != nil {
			radarrContainer.Terminate(ctx)
		}
		if sonarrContainer != nil {
			sonarrContainer.Terminate(ctx)
		}
		if prowlarrContainer != nil {
			prowlarrContainer.Terminate(ctx)
		}
		cancel()
	})

	Context("Radarr API", func() {
		It("should respond to system/status endpoint", func() {
			url := fmt.Sprintf("%s/api/v3/system/status", radarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", radarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var status map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&status)
			Expect(err).NotTo(HaveOccurred())
			Expect(status["appName"]).To(Equal("Radarr"))
		})

		It("should list quality profiles", func() {
			url := fmt.Sprintf("%s/api/v3/qualityprofile", radarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", radarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var profiles []map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&profiles)
			Expect(err).NotTo(HaveOccurred())
			// Radarr should have default quality profiles
			Expect(len(profiles)).To(BeNumerically(">", 0))
		})

		It("should list root folders (initially empty)", func() {
			url := fmt.Sprintf("%s/api/v3/rootfolder", radarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", radarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Context("Sonarr API", func() {
		It("should respond to system/status endpoint", func() {
			url := fmt.Sprintf("%s/api/v3/system/status", sonarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", sonarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var status map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&status)
			Expect(err).NotTo(HaveOccurred())
			Expect(status["appName"]).To(Equal("Sonarr"))
		})

		It("should list quality profiles", func() {
			url := fmt.Sprintf("%s/api/v3/qualityprofile", sonarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", sonarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var profiles []map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&profiles)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(profiles)).To(BeNumerically(">", 0))
		})
	})

	Context("Prowlarr API", func() {
		It("should respond to system/status endpoint", func() {
			url := fmt.Sprintf("%s/api/v1/system/status", prowlarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", prowlarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var status map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&status)
			Expect(err).NotTo(HaveOccurred())
			Expect(status["appName"]).To(Equal("Prowlarr"))
		})

		It("should list indexers (initially empty)", func() {
			url := fmt.Sprintf("%s/api/v1/indexer", prowlarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", prowlarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var indexers []map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&indexers)
			Expect(err).NotTo(HaveOccurred())
			// Fresh install should have no indexers
			Expect(len(indexers)).To(Equal(0))
		})
	})

	Context("Operator CRD Reconciliation", Ordered, func() {
		var radarrSecretName = "radarr-api-key"
		var radarrConfigName = "test-radarrconfig"

		BeforeAll(func() {
			// Skip if operator is not deployed
			cmd := exec.Command("kubectl", "get", "deployment", "nebularr-controller-manager", "-n", namespace)
			_, err := utils.Run(cmd)
			if err != nil {
				Skip("Operator not deployed, skipping CRD reconciliation tests")
			}
		})

		It("should create API key secret for Radarr", func() {
			By("creating the Radarr API key secret")
			cmd := exec.Command("kubectl", "create", "secret", "generic", radarrSecretName,
				"--from-literal=apiKey="+radarrContainer.APIKey,
				"-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reconcile a RadarrConfig CR", func() {
			By("creating a RadarrConfig CR that points to the real Radarr instance")
			radarrConfigYAML := fmt.Sprintf(`
apiVersion: arr.nebularr.io/v1alpha1
kind: RadarrConfig
metadata:
  name: %s
  namespace: %s
spec:
  instanceUrl: %s
  apiKeySecretRef:
    name: %s
    key: apiKey
  qualityProfiles:
    - name: "E2E Test Profile"
      upgradeAllowed: true
      cutoff: "Bluray-1080p"
      items:
        - name: "Bluray-1080p"
          allowed: true
`, radarrConfigName, namespace, radarrContainer.URL(), radarrSecretName)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = nil
			// Write YAML to a temp file instead
			tmpFile := "/tmp/radarrconfig-e2e.yaml"
			writeCmd := exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, radarrConfigYAML))
			_, err := utils.Run(writeCmd)
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("kubectl", "apply", "-f", tmpFile)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the RadarrConfig to be reconciled")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "radarrconfig", radarrConfigName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "RadarrConfig should be Ready")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying the quality profile was created in Radarr")
			url := fmt.Sprintf("%s/api/v3/qualityprofile", radarrContainer.URL())
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("X-Api-Key", radarrContainer.APIKey)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var profiles []map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&profiles)
			Expect(err).NotTo(HaveOccurred())

			// Find our profile
			found := false
			for _, profile := range profiles {
				if name, ok := profile["name"].(string); ok && name == "E2E Test Profile" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "E2E Test Profile should exist in Radarr")
		})

		AfterAll(func() {
			By("cleaning up test resources")
			cmd := exec.Command("kubectl", "delete", "radarrconfig", radarrConfigName, "-n", namespace, "--ignore-not-found")
			utils.Run(cmd)
			cmd = exec.Command("kubectl", "delete", "secret", radarrSecretName, "-n", namespace, "--ignore-not-found")
			utils.Run(cmd)
		})
	})
})

// TransmissionIntegration tests the DownloadStackConfig with a real Transmission instance
var _ = Describe("Transmission Integration", Ordered, Label("integration"), func() {
	var (
		ctx                   context.Context
		cancel                context.CancelFunc
		transmissionContainer *containers.TransmissionContainer
	)

	BeforeAll(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)

		By("starting Transmission container")
		var err error
		transmissionContainer, err = containers.StartTransmission(ctx, containers.TransmissionOptions{
			StartupTimeout: 2 * time.Minute,
		})
		Expect(err).NotTo(HaveOccurred(), "Failed to start Transmission container")
		GinkgoWriter.Printf("Transmission started at %s\n", transmissionContainer.URL())
	})

	AfterAll(func() {
		By("cleaning up Transmission container")
		if transmissionContainer != nil {
			transmissionContainer.Terminate(ctx)
		}
		cancel()
	})

	Context("Transmission RPC", func() {
		It("should respond to session-get request", func() {
			// Transmission RPC requires session ID, so we first get a 409 to obtain it
			req, err := http.NewRequestWithContext(ctx, "POST", transmissionContainer.URL(), nil)
			Expect(err).NotTo(HaveOccurred())
			req.SetBasicAuth(transmissionContainer.Username, transmissionContainer.Password)

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// 409 is expected on first request (need to extract session ID)
			if resp.StatusCode == http.StatusConflict {
				sessionID := resp.Header.Get("X-Transmission-Session-Id")
				Expect(sessionID).NotTo(BeEmpty(), "Should receive session ID")
			} else {
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			}
		})
	})
})
