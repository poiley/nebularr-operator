// Package discovery provides auto-discovery capabilities for *arr applications.
// This includes parsing API keys from config.xml files and inferring download client types.
package discovery

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArrConfig represents the parsed config.xml structure for *arr applications.
// All *arr apps use a similar config.xml format.
type ArrConfig struct {
	XMLName xml.Name `xml:"Config"`
	ApiKey  string   `xml:"ApiKey"`
	Port    int      `xml:"Port"`
	UrlBase string   `xml:"UrlBase"`
}

// ParseConfigXML parses an *arr config.xml file and extracts configuration.
func ParseConfigXML(r io.Reader) (*ArrConfig, error) {
	var config ArrConfig
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config.xml: %w", err)
	}
	return &config, nil
}

// ParseConfigXMLFromFile parses an *arr config.xml file from a file path.
func ParseConfigXMLFromFile(path string) (*ArrConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config.xml: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ParseConfigXML(f)
}

// PVCDiscoveryConfig holds configuration for PVC-based API key discovery.
type PVCDiscoveryConfig struct {
	// Timeout for the discovery operation (default: 60s)
	Timeout time.Duration

	// Image to use for the discovery pod (default: "busybox:1.36")
	Image string

	// TTLSecondsAfterFinished for Job cleanup (default: 60)
	TTLSecondsAfterFinished int32
}

// DefaultPVCDiscoveryConfig returns the default configuration.
func DefaultPVCDiscoveryConfig() *PVCDiscoveryConfig {
	return &PVCDiscoveryConfig{
		Timeout:                 60 * time.Second,
		Image:                   "busybox:1.36",
		TTLSecondsAfterFinished: 60,
	}
}

// DiscoverAPIKeyFromPVC attempts to discover the API key from a config.xml
// in a PersistentVolumeClaim by creating a temporary Job that reads the file.
//
// Parameters:
//   - ctx: context for the operation
//   - k8sClient: controller-runtime client for creating/watching resources
//   - clientset: kubernetes clientset for reading pod logs
//   - namespace: namespace where the PVC exists
//   - pvcName: name of the PVC containing the config
//   - configPath: path within the PVC to config.xml (default: "config.xml")
//   - config: optional configuration (uses defaults if nil)
//
// Returns the API key or an error if discovery fails.
func DiscoverAPIKeyFromPVC(
	ctx context.Context,
	k8sClient client.Client,
	clientset kubernetes.Interface,
	namespace, pvcName, configPath string,
	config *PVCDiscoveryConfig,
) (string, error) {
	if config == nil {
		config = DefaultPVCDiscoveryConfig()
	}

	if configPath == "" {
		configPath = "config.xml"
	}

	// Generate a unique job name
	jobName := fmt.Sprintf("nebularr-apikey-discovery-%s", uuid.New().String()[:8])

	// Create the Job
	job := buildDiscoveryJob(jobName, namespace, pvcName, configPath, config)

	if err := k8sClient.Create(ctx, job); err != nil {
		return "", fmt.Errorf("failed to create discovery job: %w", err)
	}

	// Ensure cleanup
	defer func() {
		// Best-effort cleanup - Job TTL should handle it if this fails
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = k8sClient.Delete(cleanupCtx, job, client.PropagationPolicy(metav1.DeletePropagationBackground))
	}()

	// Wait for Job completion
	if err := waitForJobCompletion(ctx, k8sClient, namespace, jobName, config.Timeout); err != nil {
		return "", fmt.Errorf("discovery job failed: %w", err)
	}

	// Get the pod created by the Job
	podName, err := getJobPodName(ctx, k8sClient, namespace, jobName)
	if err != nil {
		return "", fmt.Errorf("failed to find discovery pod: %w", err)
	}

	// Read pod logs
	xmlContent, err := readPodLogs(ctx, clientset, namespace, podName)
	if err != nil {
		return "", fmt.Errorf("failed to read pod logs: %w", err)
	}

	// Parse XML to extract API key
	arrConfig, err := ParseConfigXML(bytes.NewReader(xmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse config.xml from PVC: %w", err)
	}

	if arrConfig.ApiKey == "" {
		return "", fmt.Errorf("ApiKey is empty in config.xml from PVC %s/%s", namespace, pvcName)
	}

	return arrConfig.ApiKey, nil
}

// buildDiscoveryJob creates a Job spec for reading config.xml from a PVC.
func buildDiscoveryJob(name, namespace, pvcName, configPath string, config *PVCDiscoveryConfig) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "nebularr",
				"app.kubernetes.io/component":  "apikey-discovery",
				"app.kubernetes.io/managed-by": "nebularr-operator",
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr.To(config.TTLSecondsAfterFinished),
			BackoffLimit:            ptr.To(int32(0)), // No retries
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "reader",
							Image:   config.Image,
							Command: []string{"cat", "/config/" + configPath},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *parseQuantityOrPanic("100m"),
									corev1.ResourceMemory: *parseQuantityOrPanic("64Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *parseQuantityOrPanic("10m"),
									corev1.ResourceMemory: *parseQuantityOrPanic("16Mi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		},
	}
}

// parseQuantityOrPanic parses a resource quantity string and panics on error.
// This is safe because we're using hardcoded valid values.
func parseQuantityOrPanic(s string) *resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("invalid resource quantity %q: %v", s, err))
	}
	return &q
}

// waitForJobCompletion waits for a Job to complete (succeed or fail).
func waitForJobCompletion(ctx context.Context, k8sClient client.Client, namespace, jobName string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		job := &batchv1.Job{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: jobName}, job); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil // Job not found yet, keep waiting
			}
			return false, err
		}

		// Check for completion
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
				return true, nil
			}
			if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
				return false, fmt.Errorf("job failed: %s", condition.Message)
			}
		}

		return false, nil
	})
}

// getJobPodName finds the pod created by a Job.
func getJobPodName(ctx context.Context, k8sClient client.Client, namespace, jobName string) (string, error) {
	podList := &corev1.PodList{}
	if err := k8sClient.List(ctx, podList,
		client.InNamespace(namespace),
		client.MatchingLabels{"job-name": jobName},
	); err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}

	return podList.Items[0].Name, nil
}

// readPodLogs reads the logs from a pod.
func readPodLogs(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) ([]byte, error) {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	logs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, logs); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DiscoverAPIKeyFromConfigMap attempts to discover the API key from a ConfigMap.
// This is useful when the config.xml is stored in a ConfigMap (less common).
func DiscoverAPIKeyFromConfigMap(ctx context.Context, k8sClient client.Client, namespace, configMapName, key string) (string, error) {
	var cm corev1.ConfigMap
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapName}, &cm); err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, configMapName, err)
	}

	data, ok := cm.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in ConfigMap %s/%s", key, namespace, configMapName)
	}

	config, err := ParseConfigXML(strings.NewReader(data))
	if err != nil {
		return "", err
	}

	if config.ApiKey == "" {
		return "", fmt.Errorf("ApiKey is empty in config.xml from ConfigMap %s/%s", namespace, configMapName)
	}

	return config.ApiKey, nil
}

// InferDownloadClientType attempts to infer the download client type from its name.
// This is used when the user doesn't explicitly specify the implementation type.
//
// Examples:
//   - "qbittorrent" -> "qbittorrent"
//   - "my-qbit-client" -> "qbittorrent"
//   - "transmission-vpn" -> "transmission"
//   - "sabnzbd-main" -> "sabnzbd"
func InferDownloadClientType(name string) string {
	name = strings.ToLower(name)

	// Torrent clients
	torrentClients := map[string]string{
		"qbittorrent":  "qbittorrent",
		"qbit":         "qbittorrent",
		"transmission": "transmission",
		"deluge":       "deluge",
		"rtorrent":     "rtorrent",
		"rutorrent":    "rtorrent",
		"vuze":         "vuze",
		"utorrent":     "utorrent",
		"aria2":        "aria2",
		"flood":        "flood",
	}

	// Usenet clients
	usenetClients := map[string]string{
		"sabnzbd":     "sabnzbd",
		"sab":         "sabnzbd",
		"nzbget":      "nzbget",
		"nzbvortex":   "nzbvortex",
		"pneumatic":   "pneumatic",
		"usenetblack": "usenetblackhole",
	}

	// Check for matches
	for key, clientType := range torrentClients {
		if strings.Contains(name, key) {
			return clientType
		}
	}

	for key, clientType := range usenetClients {
		if strings.Contains(name, key) {
			return clientType
		}
	}

	// Unable to infer
	return ""
}

// InferProtocolFromClientType returns the protocol (torrent/usenet) for a client type.
func InferProtocolFromClientType(clientType string) string {
	torrentClients := []string{
		"qbittorrent", "transmission", "deluge", "rtorrent",
		"vuze", "utorrent", "aria2", "flood",
	}

	usenetClients := []string{
		"sabnzbd", "nzbget", "nzbvortex", "pneumatic", "usenetblackhole",
	}

	clientType = strings.ToLower(clientType)

	for _, tc := range torrentClients {
		if clientType == tc {
			return "torrent"
		}
	}

	for _, uc := range usenetClients {
		if clientType == uc {
			return "usenet"
		}
	}

	return ""
}

// DefaultConfigPaths returns the default paths where *arr apps store their config.xml
// relative to the data directory.
func DefaultConfigPaths() []string {
	return []string{
		"config.xml",
		filepath.Join("config", "config.xml"),
	}
}
