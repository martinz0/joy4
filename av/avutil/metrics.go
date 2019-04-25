package avutil

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PacketLabelValue struct {
	Who string
	Op  string
}

func clearMetrics(labels prometheus.Labels) {
	frames.Delete(labels)
	bits.Delete(labels)
	totalbits.Delete(labels)
	keyframe.Delete(labels)
	firstkeyframe.Delete(labels)
	vseqhdr.Delete(labels)
	aseqhdr.Delete(labels)
	waitfirstkeyframe.Delete(labels)
	framelatency.Delete(labels)
}

var (
	constLabels = []string{"who", "op"}
	frames      = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch fps",
		Name:      "frames",
	}, constLabels)
	bits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch video bitrate",
		Name:      "bits",
	}, constLabels)
	totalbits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch total bitrate",
		Name:      "totalbits",
	}, constLabels)
	vseqhdr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch video sequence header appears",
		Name:      "vseqhdr",
	}, constLabels)
	aseqhdr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch audio sequence header appears",
		Name:      "aseqhdr",
	}, constLabels)
	keyframe = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "help watch gop length",
		Name:      "keyframe",
	}, constLabels)
	firstkeyframe = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "first key frame",
		Name:      "firstkeyframe",
	}, constLabels)
	waitfirstkeyframe = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "interval duration bettween first I/P frame",
		Name:      "waitfirstkeyframe",
	}, constLabels)
	framelatency = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "streamer",
		Subsystem: "server",
		Help:      "interval duration bettween frame reach at and send out",
		Name:      "framelatency",
	}, constLabels)
)

func init() {
	prometheus.MustRegister(frames)
	prometheus.MustRegister(bits)
	prometheus.MustRegister(totalbits)
	prometheus.MustRegister(keyframe)
	prometheus.MustRegister(firstkeyframe)
	prometheus.MustRegister(vseqhdr)
	prometheus.MustRegister(aseqhdr)
	prometheus.MustRegister(waitfirstkeyframe)
	prometheus.MustRegister(framelatency)
}
