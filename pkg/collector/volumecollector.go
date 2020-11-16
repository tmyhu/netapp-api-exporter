package collector

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/netapp-api-exporter/pkg/netapp"
	log "github.com/sirupsen/logrus"
)

type VolumeCollector struct {
	filerName       string
	client          *netapp.Client
	metrics         []VolumeMetric
	volumes         []*netapp.Volume
	mux             sync.Mutex
	retentionPeriod time.Duration
	errorCh         chan<- error
}

type VolumeMetric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
	getterFn  func(volume *netapp.Volume) float64
}

func NewVolumeCollector(filerName string, client *netapp.Client, ch chan<- error, retentionPeriod time.Duration) *VolumeCollector {
	volumeLabels := []string{"vserver", "volume", "project_id", "share_id", "share_name", "share_type"}
	return &VolumeCollector{
		filerName:       filerName,
		client:          client,
		errorCh:         ch,
		retentionPeriod: retentionPeriod,
		metrics: []VolumeMetric{
			{
				desc: prometheus.NewDesc(
					"netapp_volume_state",
					"Netapp Volume Metrics: state (1: online; 2: restricted; 3: offline; 4: quiesced)",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return float64(v.State) },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_total_bytes",
					"Netapp Volume Metrics: total size",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SizeTotal },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_used_bytes",
					"Netapp Volume Metrics: used size",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SizeUsed },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_available_bytes",
					"Netapp Volume Metrics: available size",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SizeAvailable },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_snapshot_used_bytes",
					"Netapp Volume Metrics: size used by snapshots",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SizeUsedBySnapshots },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_snapshot_available_bytes",
					"Netapp Volume Metrics: size available for snapshots",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SizeAvailableForSnapshots },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_snapshot_reserved_bytes",
					"Netapp Volume Metrics: size reserved for snapshots",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.SnapshotReserveSize },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_used_percentage",
					"Netapp Volume Metrics: used percentage ",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.PercentageSizeUsed },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_saved_total_percentage",
					"Netapp Volume Metrics: percentage of space compression and deduplication saved",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.PercentageTotalSpaceSaved },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_saved_compression_percentage",
					"Netapp Volume Metrics: percentage of space compression saved",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.PercentageCompressionSpaceSaved },
			}, {
				desc: prometheus.NewDesc(
					"netapp_volume_saved_deduplication_percentage",
					"Netapp Volume Metrics: percentage of space deduplication saved",
					volumeLabels,
					nil),
				valueType: prometheus.GaugeValue,
				getterFn:  func(v *netapp.Volume) float64 { return v.PercentageDeduplicationSpaceSaved },
			},
		},
	}
}

func (c *VolumeCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.metrics {
		ch <- m.desc
	}
}

func (c *VolumeCollector) Collect(ch chan<- prometheus.Metric) {
	defer c.mux.Unlock()
	c.mux.Lock()

	// fetch volumes
	if c.volumes == nil {
		c.volumes = c.Fetch()
		if len(c.volumes) > 0 {
			time.AfterFunc(c.retentionPeriod, func() {
				defer c.mux.Unlock()
				c.mux.Lock()
				log.Debugf("VolumeCollector[%v] cached volumes cleared", c.filerName)
				c.volumes = nil
			})
		}
	}

	// export metrics
	log.Debugf("VolumeCollector[%v] Collect() exporting %d volumes", c.filerName, len(c.volumes))
	for _, volume := range c.volumes {
		volumeLabels := []string{volume.Vserver, volume.Volume, volume.ProjectID, volume.ShareID, volume.ShareName, volume.ShareType}
		for _, m := range c.metrics {
			ch <- prometheus.MustNewConstMetric(m.desc, m.valueType, m.getterFn(volume), volumeLabels...)
		}
	}
	return
}

func (c *VolumeCollector) Fetch() []*netapp.Volume {
	log.Debugf("VolumeCollector[%v] starts fetching volumes", c.filerName)
	volumes, err := c.client.ListVolumes()
	if err != nil {
		log.Error(err)
		return nil
	}
	log.Debugf("VolumeCollector[%v] fetched %d volumes", c.filerName, len(volumes))
	return volumes
}