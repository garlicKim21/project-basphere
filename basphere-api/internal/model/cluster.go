package model

import "time"

// ClusterStatus represents the status of a Kubernetes cluster
type ClusterStatus string

const (
	ClusterStatusPending      ClusterStatus = "pending"
	ClusterStatusProvisioning ClusterStatus = "provisioning"
	ClusterStatusReady        ClusterStatus = "ready"
	ClusterStatusDeleting     ClusterStatus = "deleting"
	ClusterStatusFailed       ClusterStatus = "failed"
)

// Cluster represents a Kubernetes cluster
type Cluster struct {
	Name              string        `json:"name"`
	Owner             string        `json:"owner"`
	Type              string        `json:"type"`                          // dev, standard
	K8sVersion        string        `json:"k8s_version"`
	ControlPlaneCount int           `json:"control_plane_count"`
	WorkerCount       int           `json:"worker_count"`
	WorkerSpec        string        `json:"worker_spec"`                   // small, medium, large
	ControlPlaneIP    string        `json:"control_plane_ip"`
	WorkerIPs         []string      `json:"worker_ips"`
	Status            ClusterStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	ReadyAt           *time.Time    `json:"ready_at,omitempty"`
	KubeconfigPath    string        `json:"kubeconfig_path,omitempty"`
}

// CreateClusterInput represents the input for creating a cluster
type CreateClusterInput struct {
	Name       string `json:"name"`
	Type       string `json:"type"`        // dev, standard
	WorkerSpec string `json:"worker_spec"` // small, medium, large
}

// Validate validates the cluster creation input
func (c *CreateClusterInput) Validate() []string {
	var errors []string

	if c.Name == "" {
		errors = append(errors, "name is required")
	} else if !isValidClusterName(c.Name) {
		errors = append(errors, "name must be 1-30 characters, lowercase letters, numbers, and hyphens only")
	}

	if c.Type == "" {
		errors = append(errors, "type is required")
	} else if !isValidClusterType(c.Type) {
		errors = append(errors, "type must be one of: dev, standard")
	}

	if c.WorkerSpec == "" {
		errors = append(errors, "worker_spec is required")
	} else if !isValidWorkerSpec(c.WorkerSpec) {
		errors = append(errors, "worker_spec must be one of: small, medium, large")
	}

	return errors
}

// isValidClusterName checks if cluster name is valid (same rules as VM name)
func isValidClusterName(name string) bool {
	if len(name) < 1 || len(name) > 30 {
		return false
	}
	for i, c := range name {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(name)-1 {
			continue
		}
		return false
	}
	return true
}

// isValidClusterType checks if cluster type is valid
func isValidClusterType(t string) bool {
	validTypes := []string{"dev", "standard"}
	for _, v := range validTypes {
		if t == v {
			return true
		}
	}
	return false
}

// isValidWorkerSpec checks if worker spec is valid
func isValidWorkerSpec(spec string) bool {
	validSpecs := []string{"small", "medium", "large"}
	for _, v := range validSpecs {
		if spec == v {
			return true
		}
	}
	return false
}

// DeleteClusterInput represents the input for deleting a cluster
type DeleteClusterInput struct {
	Force bool `json:"force,omitempty"`
}

// ClusterListResponse represents the response for listing clusters
type ClusterListResponse struct {
	Clusters []Cluster    `json:"clusters"`
	Total    int          `json:"total"`
	Quota    ClusterQuota `json:"quota"`
}

// ClusterQuota represents user's cluster quota
type ClusterQuota struct {
	MaxClusters        int `json:"max_clusters"`
	UsedClusters       int `json:"used_clusters"`
	MaxNodesPerCluster int `json:"max_nodes_per_cluster"`
}

// CreateClusterResponse represents the response for creating a cluster
type CreateClusterResponse struct {
	Cluster *Cluster `json:"cluster,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// ClusterStatusResponse represents the response for cluster status
type ClusterStatusResponse struct {
	Name   string        `json:"name"`
	Status ClusterStatus `json:"status"`
	Phase  string        `json:"phase,omitempty"`
	Nodes  []NodeStatus  `json:"nodes,omitempty"`
}

// NodeStatus represents the status of a node in the cluster
type NodeStatus struct {
	Name   string `json:"name"`
	Role   string `json:"role"`   // control-plane, worker
	Status string `json:"status"` // Ready, NotReady, Provisioning
	IP     string `json:"ip,omitempty"`
}

// KubeconfigResponse represents the response for kubeconfig
type KubeconfigResponse struct {
	Kubeconfig string `json:"kubeconfig"`
}
