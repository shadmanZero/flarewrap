package flarewrap


type Machine struct {
	CPUCores    int    `json:"cpu_cores"`
	MemoryMB    int    `json:"memory_mb"`
	StorageMB   int    `json:"storage_mb"`
	StorageType string `json:"storage_type"`
	Name        string `json:"name"`
	Image       string `json:"image"`
}
