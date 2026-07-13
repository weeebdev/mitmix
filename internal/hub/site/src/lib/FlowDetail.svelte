<script lang="ts">
  import { getFlowDetail, createRule, deleteRule, listRules, fmtDate } from '../api'
  import type { FlowDetail, Flow, Rule } from '../api'

  let {
    flow, onclose, onNavigate = (_: any) => {}
  }: {
    flow: Flow; onclose: () => void; onNavigate?: (nav: { tab: string; params?: Record<string, string> }) => void
  } = $props()

  let detail = $state<FlowDetail | null>(null)
  let loading = $state(true)
  let error = $state('')
  let tab = $state('overview')
  let copied = $state(false)
  let showRuleForm = $state(false)
  let ruleCreated = $state(false)
  let creating = $state(false)
  let quickActionMsg = $state('')
  let quickActionTimeout: number

  async function quickAction(action: string) {
    if (!detail) return
    const host = detail.host.replace(/:.*/, '')
    const domain = '*.' + host.split('.').slice(-2).join('.')
    try {
      const all = await listRules()
      const existing = all.find(r => {
        if (!r.enabled || r.action !== action) return false
        try { return JSON.parse(r.match).host === domain } catch { return false }
      })
      if (existing) {
        await deleteRule(existing.id)
        quickActionMsg = action === 'intercept' ? 'Intercept OFF for ' + domain : 'Decrypt OFF for ' + domain
      } else {
        await createRule({
          action, match: JSON.stringify({ host: domain }),
          priority: 100, enabled: true,
        } as any)
        quickActionMsg = action === 'intercept' ? 'Intercept ON for ' + domain : 'Decrypt ON for ' + domain
      }
      clearTimeout(quickActionTimeout)
      quickActionTimeout = setTimeout(() => quickActionMsg = '', 3000)
    } catch (e: any) {
      alert('Failed: ' + e.message)
    }
  }

  $effect(() => {
    loading = true
    error = ''
    getFlowDetail(flow.id).then(d => { detail = d; loading = false }).catch(e => { error = e.message; loading = false })
  })

  function copyCurl() {
    if (!detail) return
    let cmd = `curl -X ${detail.method}`
    if (detail.req_headers) {
      for (const [k, vals] of Object.entries(detail.req_headers)) {
        const h = Array.isArray(vals) ? vals.join(', ') : vals
        cmd += ` -H '${k}: ${h}'`
      }
    }
    if (detail.req_body) {
      cmd += ` -d '${detail.req_body.replace(/'/g, "\\'")}'`
    }
    cmd += ` 'http://${detail.host}${detail.path}'`
    navigator.clipboard.writeText(cmd)
    copied = true
    setTimeout(() => copied = false, 2000)
  }

  async function writeRule() {
    if (!detail) return
    creating = true
    try {
      const match = JSON.stringify({ host: detail.host, path: detail.path, method: detail.method })
      await createRule({ node: '*', priority: 0, action: 'record', match, spec: '{}', enabled: true })
      ruleCreated = true
      showRuleForm = false
      setTimeout(() => ruleCreated = false, 3000)
    } catch (e: any) {
      alert('Failed: ' + e.message)
    } finally {
      creating = false
    }
  }
</script>

<div class="panel">
  <div class="panel-header">
    <h3>Flow Detail</h3>
    <div class="header-actions">
      <button class="btn-action" onclick={() => quickAction('intercept')}>⏸ Intercept</button>
      <button class="btn-action" onclick={copyCurl}>{copied ? 'Copied!' : 'cURL'}</button>
      <button class="btn-action rule" onclick={() => showRuleForm = !showRuleForm}>
        {showRuleForm ? 'Cancel' : 'Rule'}
      </button>
      <button class="close" onclick={onclose}>&times;</button>
    </div>
  </div>

  {#if loading}
    <div class="loading">Loading...</div>
  {:else if error}
    <div class="error">{error}</div>
  {:else if detail}
    <div class="tabs">
      <button class:active={tab === 'overview'} onclick={() => tab = 'overview'}>Overview</button>
      <button class:active={tab === 'request'} onclick={() => tab = 'request'}>Request</button>
      <button class:active={tab === 'response'} onclick={() => tab = 'response'}>Response</button>
    </div>

    <div class="tab-content">
      {#if tab === 'overview'}
        <div class="field"><label>Method</label><span class="mono">{detail.method}</span></div>
        <div class="field"><label>Host</label><span>{detail.host}</span></div>
        <div class="field"><label>Path</label><span class="mono">{detail.path}</span></div>
        <div class="field"><label>Status</label><span>{detail.status_code}</span></div>
        <div class="field"><label>Duration</label><span>{detail.duration_ms}ms</span></div>
        <div class="field"><label>Req Size</label><span>{detail.req_size} bytes</span></div>
        <div class="field"><label>Resp Size</label><span>{detail.resp_size} bytes</span></div>
        <div class="field"><label>Captured</label><span>{fmtDate(detail.captured_at)}</span></div>
        {#if detail.app_name}
          <div class="field">
            <label>App</label>
            <button class="link" onclick={() => onNavigate({ tab: 'flows', params: { app: detail.app_name } })}>{detail.app_name}</button>
          </div>
        {/if}
        {#if detail.source_host}
          <div class="field">
            <label>Source</label>
            <button class="link" onclick={() => onNavigate({ tab: 'flows', params: { source: detail.source_host } })}>{detail.source_host}</button>
          </div>
        {/if}
      {:else if tab === 'request'}
        <h4>Headers</h4>
        <div class="headers">
          {#if detail.req_headers && Object.keys(detail.req_headers).length}
            {#each Object.entries(detail.req_headers) as [key, vals]}
              <div class="header-row"><span class="hdr-key">{key}:</span><span class="mono">{vals}</span></div>
            {/each}
          {:else}
            <span class="empty">No headers</span>
          {/if}
        </div>
        <h4>Body</h4>
        <pre class="body">{detail.req_body || '(empty)'}</pre>
      {:else if tab === 'response'}
        <h4>Headers</h4>
        <div class="headers">
          {#if detail.resp_headers && Object.keys(detail.resp_headers).length}
            {#each Object.entries(detail.resp_headers) as [key, vals]}
              <div class="header-row"><span class="hdr-key">{key}:</span><span class="mono">{vals}</span></div>
            {/each}
          {:else}
            <span class="empty">No headers</span>
          {/if}
        </div>
        <h4>Body</h4>
        <pre class="body">{detail.resp_body || '(empty)'}</pre>
      {/if}
    </div>

    {#if showRuleForm}
      <div class="rule-form">
        <h4>New Rule from Flow</h4>
        <div class="field"><label>Method</label><span class="mono">{detail.method}</span></div>
        <div class="field"><label>Host</label><span>{detail.host}</span></div>
        <div class="field"><label>Path</label><span class="mono">{detail.path}</span></div>
        <div class="rule-actions">
          <button onclick={writeRule} disabled={creating}>{creating ? 'Creating...' : 'Create Rule'}</button>
          <button class="btn-cancel" onclick={() => showRuleForm = false}>Cancel</button>
        </div>
      </div>
    {/if}

    {#if quickActionMsg}
      <div class="toast">{quickActionMsg}</div>
    {/if}
    {#if ruleCreated}
      <div class="toast">Rule created!</div>
    {/if}
  {/if}
</div>

<style>
  .panel { background: #161b22; border: 1px solid #30363d; border-radius: 6px; width: 480px; max-height: calc(100vh - 120px); overflow-y: auto; }
  .panel-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid #30363d; gap: 8px; }
  .panel-header h3 { font-size: 14px; color: #c9d1d9; white-space: nowrap; }
  .header-actions { display: flex; gap: 4px; align-items: center; }
  .btn-action { background: #21262d; border: 1px solid #30363d; color: #c9d1d9; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 11px; white-space: nowrap; }
  .btn-action:hover { background: #30363d; }
  .close { background: none; border: none; color: #8b949e; font-size: 20px; cursor: pointer; padding: 0 4px; line-height: 1; }
  .close:hover { color: #f85149; }
  .loading, .error { padding: 24px; text-align: center; color: #8b949e; }
  .error { color: #f85149; }
  .tabs { display: flex; border-bottom: 1px solid #30363d; }
  .tabs button { flex: 1; background: none; border: none; color: #8b949e; padding: 8px 12px; cursor: pointer; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
  .tabs button.active { color: #c9d1d9; border-bottom: 2px solid #58a6ff; }
  .tabs button:hover { color: #c9d1d9; background: #21262d; }
  .tab-content { padding: 16px; }
  .field { margin-bottom: 8px; }
  .field label { display: inline-block; width: 100px; color: #8b949e; font-size: 12px; text-transform: uppercase; }
  .field span { font-size: 13px; }
  .mono { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; }
  h4 { font-size: 11px; color: #8b949e; text-transform: uppercase; margin: 16px 0 8px; }
  .headers { margin-bottom: 8px; }
  .header-row { padding: 2px 0; font-size: 12px; }
  .hdr-key { color: #79c0ff; margin-right: 8px; }
  .body { background: #0d1117; padding: 12px; border-radius: 4px; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; line-height: 1.4; max-height: 400px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
  .empty { color: #8b949e; font-size: 12px; }
  .rule-form { padding: 12px 16px; border-top: 1px solid #30363d; background: #0d1117; }
  .rule-form h4 { margin: 0 0 8px; }
  .rule-actions { display: flex; gap: 8px; margin-top: 12px; }
  .rule-actions button { padding: 6px 16px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; }
  .rule-actions button:hover { background: #2ea043; }
  .rule-actions button:disabled { opacity: 0.6; cursor: default; }
  .btn-cancel { background: #21262d !important; color: #c9d1d9 !important; border: 1px solid #30363d !important; }
  .btn-cancel:hover { background: #30363d !important; }
  .toast { position: sticky; bottom: 0; padding: 8px 16px; background: #238636; color: #fff; font-size: 12px; text-align: center; }
  .link { background: none; border: none; color: #58a6ff; cursor: pointer; padding: 0; font: inherit; font-size: 13px; text-decoration: underline; }
  .link:hover { color: #79c0ff; }
</style>
