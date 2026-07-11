package hub

import (
	"log"
	"os"
	"strconv"
	"time"
)

func retentionHours() int {
	s := os.Getenv("FLOW_RETENTION_HOURS")
	if s == "" {
		return 24
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 24
	}
	return n
}

func (h *Hub) startRetention() {
	hours := retentionHours()
	interval := time.Duration(hours) * time.Hour / 2
	if interval < time.Hour {
		interval = time.Hour
	}
	log.Printf("flow retention: deleting flows older than %dh (check every %v)", hours, interval)

	go func() {
		for {
			time.Sleep(interval)
			h.runRetention(hours)
		}
	}()
}

func (h *Hub) runRetention(hours int) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).UTC().Format(time.RFC3339)

	oldFlows, err := h.FindRecordsByFilter("flows", "captured_at < {:cutoff}", "", 0, 0, map[string]any{"cutoff": cutoff})
	if err != nil {
		log.Printf("retention query error: %v", err)
		return
	}

	if len(oldFlows) == 0 {
		return
	}

	for _, rec := range oldFlows {
		id := rec.GetString("id")
		h.Delete(rec)
		h.FindRecordsByFilter("flow_bodies", "flow = {:flow}", "", 0, 0, map[string]any{"flow": id})
		bodyRecs, _ := h.FindRecordsByFilter("flow_bodies", "flow = {:flow}", "", 0, 0, map[string]any{"flow": id})
		for _, b := range bodyRecs {
			h.Delete(b)
		}
	}

	log.Printf("retention: deleted %d flows older than %dh", len(oldFlows), hours)
}
