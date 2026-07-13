<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { listFlows, fmtTime, connectLive, getStats, listRules, createRule, deleteRule } from '../api'
  import type { Flow, Rule } from '../api'
  import FlowDetail from './FlowDetail.svelte'

  let {
    flowFilters = {},
    onNavigate = (_: any) => {},
  }: {
    flowFilters?: { host?: string; method?: string; status?: string; app?: string; source?: string }
    onNavigate?: (nav: { tab: string; params?: Record<string, string> }) => void
  } = $props()

  let flows = $state<Flow[]>([])
  let selected = $state<Flow | null>(null)
  let error = $state('')
  let newFlowIds = $state<Set<string>>(new Set())
  let topApps = $state<string[]>([])
  let decryptRules = $state<Map<string, string>>(new Map())
  let interceptRules = $state<Map<string, string>>(new Map())
  let rulesLoaded = $state(false)
  let interval: number
  let cleanup: (() => void) | null = null
  let sortKey = $state<string>('captured_at')
  let sortDir = $state<-1 | 1>(-1)

  let method = $state('')
  let host = $state('')
  let status = $state('')
  let app = $state('')
  let source = $state('')
  let filterMethod = $state('')
  let filterHost = $state('')
  let filterStatus = $state('')
  let filterApp = $state('')
  let filterSource = $state('')

  function getFilterParams(): { host?: string; method?: string; status?: string; app?: string; source?: string } {
    const p: { host?: string; method?: string; status?: string; app?: string; source?: string } = {}
    if (filterMethod) p.method = filterMethod
    if (filterHost) p.host = filterHost
    if (filterStatus) p.status = filterStatus
    if (filterApp) p.app = filterApp
    if (filterSource) p.source = filterSource
    return p
  }

  async function load() {
    try { flows = await listFlows(getFilterParams()) }
    catch (e: any) { error = e.message }
  }

  async function loadRules() {
    try {
      const all = await listRules()
      const dr = new Map<string, string>()
      const ir = new Map<string, string>()
      for (const r of all) {
        if (!r.enabled) continue
        let host = ''
        try { host = JSON.parse(r.match).host || '' } catch {}
        if (!host) continue
        if (r.action === 'decrypt') dr.set(host, r.id)
        if (r.action === 'intercept') ir.set(host, r.id)
      }
      decryptRules = dr
      interceptRules = ir
      rulesLoaded = true
    } catch {}
  }

  async function toggleAction(host: string, action: string, ruleId: string | undefined) {
    if (ruleId) {
      await deleteRule(ruleId)
    } else {
      await createRule({
        action,
        match: JSON.stringify({ host: '*.' + host.split('.').slice(-2).join('.') }),
        priority: 100,
        enabled: true,
      } as any)
    }
    loadRules()
  }

  function applyFilters() {
    filterMethod = method
    filterHost = host
    filterStatus = status
    filterApp = app
    filterSource = source
    load()
  }

  function clearFilters() {
    method = ''; host = ''; status = ''; app = ''; source = ''
    filterMethod = ''; filterHost = ''; filterStatus = ''; filterApp = ''; filterSource = ''
    load()
  }

  function toggleSort(key: string) {
    if (sortKey === key) { sortDir = sortDir === 1 ? -1 : 1 }
    else { sortKey = key; sortDir = -1 }
  }

  function sortIcon(key: string): string {
    if (sortKey !== key) return '↕'
    return sortDir === 1 ? '↑' : '↓'
  }

  let sorted = $derived([...flows].sort((a: any, b: any) => {
    let av = a[sortKey], bv = b[sortKey]
    if (typeof av === 'string') av = av.toLowerCase()
    if (typeof bv === 'string') bv = bv.toLowerCase()
    if (av < bv) return -1 * sortDir
    if (av > bv) return 1 * sortDir
    return 0
  }))

  onMount(() => {
    if (flowFilters.method) { method = flowFilters.method; filterMethod = flowFilters.method }
    if (flowFilters.host) { host = flowFilters.host; filterHost = flowFilters.host }
    if (flowFilters.status) { status = flowFilters.status; filterStatus = flowFilters.status }
    if (flowFilters.app) { app = flowFilters.app; filterApp = flowFilters.app }
    if (flowFilters.source) { source = flowFilters.source; filterSource = flowFilters.source }
    load()
    getStats().then(s => { topApps = (s.top_apps || []).map(a => a.app) }).catch(() => {})
    loadRules()
    interval = setInterval(load, 5000)
    cleanup = connectLive((msg) => {
      if (msg.action === 'flow_created') {
        const f = msg.data
        flows = [f, ...flows].slice(0, 200)
        newFlowIds = new Set([f.id, ...newFlowIds])
        setTimeout(() => {
          newFlowIds = new Set([...newFlowIds].filter(id => id !== f.id))
        }, 2000)
      }
    })
  })
  onDestroy(() => {
    clearInterval(interval)
    if (cleanup) cleanup()
  })
</script>

<div class="card"><h3>Captured Flows</h3><div class="val">{flows.length}</div></div>

<div class="filter-bar">
  <select bind:value={method}>
    <option value="">All Methods</option>
    <option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option><option>PATCH</option>
  </select>
  <input type="text" placeholder="Host" bind:value={host} />
  <input type="text" placeholder="Status" bind:value={status} />
  <input type="text" placeholder="App" bind:value={app} list="app-suggestions" />
  <datalist id="app-suggestions">
    {#each topApps as a}
      <option value={a}></option>
    {/each}
  </datalist>
  <input type="text" placeholder="Source Host" bind:value={source} />
  <button onclick={applyFilters}>Apply</button>
  <button class="clear" onclick={clearFilters}>Clear</button>
</div>

<div class="flows-layout">
  <div class="flows-table">
    <table>
      <thead><tr>
        <th onclick={() => toggleSort('captured_at')} class="sort">{sortIcon('captured_at')} Time</th>
        <th onclick={() => toggleSort('method')} class="sort">{sortIcon('method')} Method</th>
        <th onclick={() => toggleSort('host')} class="sort">{sortIcon('host')} Host</th>
        <th onclick={() => toggleSort('path')} class="sort">{sortIcon('path')} Path</th>
        <th onclick={() => toggleSort('status_code')} class="sort">{sortIcon('status_code')} Status</th>
        <th onclick={() => toggleSort('app_name')} class="sort">{sortIcon('app_name')} App</th>
        <th onclick={() => toggleSort('source_host')} class="sort">{sortIcon('source_host')} Source</th>
        <th onclick={() => toggleSort('duration_ms')} class="sort">{sortIcon('duration_ms')} Dur</th>
        <th>Actions</th>
      </tr></thead>
      <tbody>
        {#if flows.length}
          {#each sorted as f}
            <tr class:selected={selected?.id === f.id} class:newflow={newFlowIds.has(f.id)} onclick={() => selected = f}>
              <td>{fmtTime(f.captured_at)}</td>
              <td>{f.method}</td><td>{f.host}</td><td>{f.path}</td>
              <td>{f.status_code}</td>
              <td>
                {#if f.app_name}
                  <button class="link" onclick={(e) => { e.stopPropagation(); onNavigate({ tab: 'flows', params: { app: f.app_name } }) }}>{f.app_name}</button>
                {:else}—{/if}
              </td>
              <td>
                {#if f.source_host}
                  <button class="link" onclick={(e) => { e.stopPropagation(); onNavigate({ tab: 'flows', params: { source: f.source_host } }) }}>{f.source_host}</button>
                {:else}—{/if}
              </td>
              <td>{f.duration_ms}ms</td>
              <td class="actions">
                {#if rulesLoaded}
                  <button class="action-btn decrypt" title="Decrypt (capture bodies)" onclick={(e) => { e.stopPropagation(); toggleAction(f.host.replace(/:.*/, ''), 'decrypt', decryptRules.get('*.' + f.host.replace(/:.*/, '').split('.').slice(-2).join('.'))) }}>
                    {decryptRules.has('*.' + f.host.replace(/:.*/, '').split('.').slice(-2).join('.')) ? '🔓' : '🔒'}
                  </button>
                  <button class="action-btn intercept" title="Intercept (pause)" onclick={(e) => { e.stopPropagation(); toggleAction(f.host.replace(/:.*/, ''), 'intercept', interceptRules.get('*.' + f.host.replace(/:.*/, '').split('.').slice(-2).join('.'))) }}>
                    {interceptRules.has('*.' + f.host.replace(/:.*/, '').split('.').slice(-2).join('.')) ? '⏸' : '⏭'}
                  </button>
                {:else}
                  <span class="loading">…</span>
                {/if}
              </td>
            </tr>
          {/each}
        {:else}
          <tr><td colspan="9" class="empty">No flows captured{error ? ': ' + error : ''}</td></tr>
        {/if}
      </tbody>
    </table>
  </div>

  {#if selected}
    <div class="detail-panel">
      <FlowDetail flow={selected} onclose={() => selected = null} {onNavigate} />
    </div>
  {/if}
</div>

<style>
  .card { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; margin-bottom: 16px; display: inline-block; }
  .card h3 { font-size: 12px; color: #8b949e; text-transform: uppercase; }
  .card .val { font-size: 28px; font-weight: 700; }
  .flows-layout { display: flex; gap: 16px; align-items: flex-start; position: relative; }
  .flows-table { flex: 1; min-width: 0; }
  .detail-panel { flex-shrink: 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  tr { cursor: pointer; }
  tr:hover { background: #161b22; }
  tr.selected { background: #1c2128; }
  tr.newflow { animation: flash 2s ease-out; }
  @keyframes flash { 0% { background: #1c3d2e; } 100% { background: transparent; } }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
  .sort { cursor: pointer; user-select: none; }
  .sort:hover { color: #c9d1d9; }
  .filter-bar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; flex-wrap: wrap; }
  .filter-bar select, .filter-bar input { padding: 4px 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 13px; }
  .filter-bar button { padding: 4px 12px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 4px; cursor: pointer; font-size: 13px; }
  .filter-bar button:hover { background: #30363d; }
  .filter-bar button.clear { color: #8b949e; }
  .link { background: none; border: none; color: #58a6ff; cursor: pointer; padding: 0; font: inherit; text-decoration: underline; }
  .link:hover { color: #79c0ff; }
  .actions { white-space: nowrap; display: flex; gap: 2px; }
  .action-btn { background: none; border: 1px solid #30363d; border-radius: 3px; cursor: pointer; font-size: 12px; padding: 1px 4px; line-height: 1.4; }
  .action-btn:hover { background: #21262d; }
  .action-btn.decrypt:hover { border-color: #58a6ff; }
  .action-btn.intercept:hover { border-color: #d29922; }
  .loading { color: #8b949e; font-size: 12px; }
</style>
