<script lang="ts">
  import { onMount } from 'svelte'
  import { getStats } from '../api'
  import type { Stats } from '../api'

  let { onNavigate = (_: any) => {} }: { onNavigate?: (nav: { tab: string; params?: Record<string, string> }) => void } = $props()

  let stats = $state<Stats | null>(null)
  let error = $state('')
  let loading = $state(true)

  onMount(() => load())

  async function load() {
    loading = true; error = ''
    try { stats = await getStats() }
    catch (e: any) { error = e.message || 'failed to load' }
    finally { loading = false }
  }

  function fmtBytes(n: number): string {
    if (!n) return '0 B'
    const u = ['B','KB','MB','GB']
    let i = 0; let v = n
    while (v >= 1024 && i < u.length-1) { v /= 1024; i++ }
    return v.toFixed(1) + ' ' + u[i]
  }

  function barWidth(val: number, max: number): string {
    return max > 0 ? (val / max * 100) + '%' : '0%'
  }

  function fmtDuration(ms: number): string {
    if (ms < 1000) return ms.toFixed(0) + 'ms'
    return (ms / 1000).toFixed(1) + 's'
  }

  let statusColors: Record<string, string> = {
    '2': '#3fb950', '3': '#d29922', '4': '#f85149', '5': '#da3633'
  }

  function statusGroup(code: string): string {
    return code[0]
  }

  function groupCodes(codes: Record<string, number>): Record<string, { label: string; count: number; codes: [string, number][] }> {
    let g: Record<string, any> = {}
    for (const [k, v] of Object.entries(codes)) {
      const p = k[0]
      if (!g[p]) g[p] = { label: p + 'xx', count: 0, codes: [] }
      g[p].count += v
      g[p].codes.push([k, v])
    }
    return g
  }

  function maxHourly(arr: { hour: string; count: number }[]): number {
    let m = 0
    for (const x of arr) { if (x.count > m) m = x.count }
    return m
  }
</script>

{#if loading}
  <p class="loading">Loading stats...</p>
{:else if error}
  <p class="error">{error}</p>
{:else if stats}
  <div class="stats-grid">
    <div class="card clickable" onclick={() => onNavigate({ tab: 'flows' })}><h3>Total Flows</h3><div class="big">{stats.total_flows}</div></div>
    <div class="card"><h3>Avg Duration</h3><div class="big">{fmtDuration(stats.duration_avg)}</div></div>
    <div class="card"><h3>Max Duration</h3><div class="big">{fmtDuration(stats.duration_max)}</div></div>
    <div class="card"><h3>Success Rate</h3><div class="big">{(stats.success_rate).toFixed(1)}%</div></div>
    <div class="card"><h3>Req Data</h3><div class="big">{fmtBytes(stats.total_req_size)}</div></div>
    <div class="card"><h3>Resp Data</h3><div class="big">{fmtBytes(stats.total_resp_size)}</div></div>
  </div>

  <div class="charts-row">
    <div class="chart-card">
      <h3>Status Codes</h3>
      <div class="bar-chart">
        {#each Object.entries(groupCodes(stats.status_codes)) as [group, g]}
          <div class="bar-row clickable" onclick={() => onNavigate({ tab: 'flows', params: { status: group + 'xx' } })}>
            <span class="bar-label" style="color: {statusColors[group] || '#8b949e'}">{g.label}</span>
            <div class="bar-track">
              <div class="bar-fill" style="width: {barWidth(g.count, stats.total_flows)}; background: {statusColors[group] || '#8b949e'}"></div>
            </div>
            <span class="bar-val">{g.count}</span>
          </div>
          {#each g.codes as [code, count]}
            <div class="bar-row sub clickable" onclick={() => onNavigate({ tab: 'flows', params: { status: code } })}>
              <span class="bar-label sub-label">{code}</span>
              <div class="bar-track">
                <div class="bar-fill" style="width: {barWidth(count, stats.total_flows)}; background: {statusColors[group] || '#8b949e'}; opacity: 0.6"></div>
              </div>
              <span class="bar-val">{count}</span>
            </div>
          {/each}
        {:else}
          <p class="empty">No data</p>
        {/each}
      </div>
    </div>

    <div class="chart-card">
      <h3>Methods</h3>
      <div class="bar-chart">
        {#each Object.entries(stats.methods) as [method, count]}
          <div class="bar-row clickable" onclick={() => onNavigate({ tab: 'flows', params: { method } })}>
            <span class="bar-label">{method}</span>
            <div class="bar-track">
              <div class="bar-fill" style="width: {barWidth(count, stats.total_flows)}; background: #58a6ff"></div>
            </div>
            <span class="bar-val">{count}</span>
          </div>
        {:else}
          <p class="empty">No data</p>
        {/each}
      </div>
    </div>
  </div>

  {#if stats.hourly.length}
    <div class="chart-card">
      <h3>Flows (Last 24h)</h3>
      <div class="hourly-chart">
        {#each stats.hourly as h}
          <div class="hour-bar" title="{h.hour}: {h.count} flows">
            <div class="hour-fill" style="height: {barWidth(h.count, maxHourly(stats.hourly))}; background: #58a6ff"></div>
            <span class="hour-label">{h.hour.slice(11, 16)}</span>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if stats.top_hosts.length}
    <div class="chart-card">
      <h3>Top Hosts</h3>
      <div class="bar-chart">
        {#each stats.top_hosts as h}
          <div class="bar-row clickable" onclick={() => onNavigate({ tab: 'flows', params: { host: h.host } })}>
            <span class="bar-label host-label">{h.host}</span>
            <div class="bar-track">
              <div class="bar-fill" style="width: {barWidth(h.count, stats.top_hosts[0].count)}; background: #7ee787"></div>
            </div>
            <span class="bar-val">{h.count}</span>
          </div>
        {/each}
      </div>
    </div>
  {/if}
{/if}

<style>
  .loading { text-align: center; padding: 48px; color: #8b949e; }
  .error { color: #f85149; text-align: center; padding: 24px; }
  .stats-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 12px; margin-bottom: 16px; }
  .card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; text-align: center; }
  .card h3 { font-size: 12px; color: #8b949e; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
  .card .big { font-size: 28px; font-weight: 700; color: #c9d1d9; }
  .card.clickable { cursor: pointer; transition: border-color 0.15s; }
  .card.clickable:hover { border-color: #58a6ff; }
  .charts-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
  .chart-card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; margin-bottom: 16px; }
  .chart-card h3 { font-size: 14px; color: #c9d1d9; margin-bottom: 12px; }
  .bar-chart { display: flex; flex-direction: column; gap: 2px; }
  .bar-row { display: flex; align-items: center; gap: 8px; padding: 2px 4px; border-radius: 3px; }
  .bar-row.sub { padding-left: 20px; }
  .bar-row.clickable { cursor: pointer; }
  .bar-row.clickable:hover { background: #1c2128; }
  .bar-label { width: 50px; font-size: 13px; color: #8b949e; text-align: right; flex-shrink: 0; }
  .bar-label.sub-label { font-size: 12px; color: #6e7681; }
  .host-label { width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .bar-track { flex: 1; height: 16px; background: #0d1117; border-radius: 3px; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 3px; transition: width 0.3s; min-width: 2px; }
  .bar-val { width: 40px; font-size: 13px; color: #c9d1d9; flex-shrink: 0; }
  .hourly-chart { display: flex; align-items: flex-end; gap: 2px; height: 120px; padding: 4px 0; }
  .hour-bar { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-end; height: 100%; }
  .hour-fill { width: 100%; max-width: 24px; border-radius: 2px 2px 0 0; transition: height 0.3s; min-height: 2px; }
  .hour-label { font-size: 9px; color: #8b949e; margin-top: 4px; }
  .empty { color: #8b949e; font-size: 13px; }
  @media (max-width: 768px) { .charts-row { grid-template-columns: 1fr; } }
</style>
