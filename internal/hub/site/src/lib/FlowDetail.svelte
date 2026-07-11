<script lang="ts">
  import { getFlowDetail } from '../api'
  import type { FlowDetail } from '../api'
  import type { Flow } from '../api'

  let { flow, onclose }: { flow: Flow; onclose: () => void } = $props()

  let detail = $state<FlowDetail | null>(null)
  let loading = $state(true)
  let error = $state('')
  let tab = $state('overview')

  $effect(() => {
    loading = true
    error = ''
    getFlowDetail(flow.id).then(d => { detail = d; loading = false }).catch(e => { error = e.message; loading = false })
  })
</script>

<div class="panel">
  <div class="panel-header">
    <h3>Flow Detail</h3>
    <button class="close" onclick={onclose}>&times;</button>
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
        <div class="field"><label>Captured</label><span>{new Date(detail.captured_at).toLocaleString()}</span></div>
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
  {/if}
</div>

<style>
  .panel { background: #161b22; border: 1px solid #30363d; border-radius: 6px; width: 480px; max-height: calc(100vh - 120px); overflow-y: auto; position: sticky; top: 0; }
  .panel-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid #30363d; }
  .panel-header h3 { font-size: 14px; color: #c9d1d9; }
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
</style>
