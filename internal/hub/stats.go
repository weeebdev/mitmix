package hub

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type StatsResponse struct {
	TotalFlows    int            `json:"total_flows"`
	DurationAvg   float64        `json:"duration_avg"`
	DurationMax   int            `json:"duration_max"`
	TotalReqSize  int64          `json:"total_req_size"`
	TotalRespSize int64          `json:"total_resp_size"`
	StatusCodes   map[string]int `json:"status_codes"`
	Methods       map[string]int `json:"methods"`
	TopHosts      []HostCount    `json:"top_hosts"`
	TopApps       []AppCount     `json:"top_apps"`
	Hourly        []HourlyCount  `json:"hourly"`
	SuccessRate   float64        `json:"success_rate"`
}

type HostCount struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

type AppCount struct {
	App   string `json:"app"`
	Count int    `json:"count"`
}

type HourlyCount struct {
	Hour  string `json:"hour"`
	Count int    `json:"count"`
}

func computeStats(h *Hub) (StatsResponse, error) {
	var s StatsResponse
	q := h.DB().
		Select("count(*)").
		From("flows")
	var total int
	if err := q.Row(&total); err != nil {
		total = 0
	}
	s.TotalFlows = total

	var avgDuration float64
	var maxDuration int
	var totalReq, totalResp int64
	h.DB().
		Select("coalesce(avg(duration_ms),0)", "coalesce(max(duration_ms),0)", "coalesce(sum(req_size),0)", "coalesce(sum(resp_size),0)").
		From("flows").
		Row(&avgDuration, &maxDuration, &totalReq, &totalResp)
	s.DurationAvg = avgDuration
	s.DurationMax = maxDuration
	s.TotalReqSize = totalReq
	s.TotalRespSize = totalResp

	statusCodes := map[string]int{}
	rows, _ := h.DB().
		Select("status_code", "count(*) as c").
		From("flows").
		GroupBy("status_code").
		Rows()
	if rows != nil {
		for rows.Next() {
			var sc int
			var c int
			rows.Scan(&sc, &c)
			statusCodes[fmt.Sprintf("%d", sc)] = c
		}
		rows.Close()
	}
	s.StatusCodes = statusCodes

	methods := map[string]int{}
	mrows, _ := h.DB().
		Select("method", "count(*) as c").
		From("flows").
		GroupBy("method").
		Rows()
	if mrows != nil {
		for mrows.Next() {
			var m string
			var c int
			mrows.Scan(&m, &c)
			methods[m] = c
		}
		mrows.Close()
	}
	s.Methods = methods

	var hosts []HostCount
	hrows, _ := h.DB().
		Select("host", "count(*) as c").
		From("flows").
		GroupBy("host").
		OrderBy("c desc").
		Limit(10).
		Rows()
	if hrows != nil {
		for hrows.Next() {
			var host string
			var c int
			hrows.Scan(&host, &c)
			hosts = append(hosts, HostCount{Host: host, Count: c})
		}
		hrows.Close()
	}
	s.TopHosts = hosts

	var apps []AppCount
	arows, _ := h.DB().
		Select("app_name", "count(*) as c").
		From("flows").
		Where(dbx.NewExp("app_name != ''")).
		GroupBy("app_name").
		OrderBy("c desc").
		Limit(20).
		Rows()
	if arows != nil {
		for arows.Next() {
			var app string
			var c int
			arows.Scan(&app, &c)
			apps = append(apps, AppCount{App: app, Count: c})
		}
		arows.Close()
	}
	s.TopApps = apps

	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	var hourly []HourlyCount
	hhrows, _ := h.DB().
		Select("strftime('%Y-%m-%dT%H:00:00Z', captured_at) as hour", "count(*) as c").
		From("flows").
		Where(dbx.NewExp("captured_at >= {:cutoff}", dbx.Params{"cutoff": cutoff})).
		GroupBy("hour").
		OrderBy("hour asc").
		Rows()
	if hhrows != nil {
		for hhrows.Next() {
			var hour string
			var c int
			hhrows.Scan(&hour, &c)
			hourly = append(hourly, HourlyCount{Hour: hour, Count: c})
		}
		hhrows.Close()
	}
	s.Hourly = hourly

	var successRate float64
	if total > 0 {
		var success int
		h.DB().
			Select("count(*)").From("flows").
			Where(dbx.NewExp("status_code >= 200 AND status_code < 400")).
			Row(&success)
		successRate = float64(success) / float64(total) * 100
	}
	s.SuccessRate = successRate

	return s, nil
}

func (h *Hub) handleStats(e *core.RequestEvent) error {
	stats, err := computeStats(h)
	if err != nil {
		return e.InternalServerError("stats failed", err)
	}
	return e.JSON(200, stats)
}


