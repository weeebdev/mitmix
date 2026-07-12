<script lang="ts">
  import { onMount } from 'svelte'
  import { listFlows, fmtDate, fmtTime } from '../api'
  import type { Node, Flow } from '../api'

  let { node, onclose }: { node: Node, onclose: () => void } = $props()

  let flows = $state<Flow[]>([])
  let error = $state('')
  let copied = $state(false)

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(node.token)
      copied = true
      setTimeout(() => copied = false, 2000)
    } catch {}
  }

  onMount(async () => {
    try { flows = await listFlows() }
    catch (e: any) { error = e.message }
  })
</script>

<div class="overlay" onclick={onclose} role="presentation"></div>
<div class="panel">
  <button class="close" onclick={onclose}>&times;</button>

  <h2>{node.name || node.id}</h2>

  <section>
    <h3>Overview</h3>
    <dl>
      <dt>Status</dt><dd class="status-{node.status}">{node.status || 'down'}</dd>
      <dt>Fingerprint</dt><dd>{node.fingerprint || '-'}</dd>
      <dt>Version</dt><dd>{node.version || '-'}</dd>
      <dt>Last Seen</dt><dd>{fmtDate(node.last_seen)}</dd>
    </dl>
  </section>

  <section>
    <h3>Token</h3>
    <div class="token-row">
      <code>{node.token?.slice(0, 16)}…{node.token?.slice(-8)}</code>
      <button class="copy-btn" onclick={copyToken}>{copied ? 'Copied!' : 'Copy'}</button>
    </div>
  </section>

  <section>
    <h3>Recent Flows ({flows.length})</h3>
    {#if flows.length}
      <table>
        <thead><tr><th>Time</th><th>Method</th><th>Host</th><th>Path</th><th>Status</th><th>Duration</th></tr></thead>
        <tbody>
          {#each flows as f}
            <tr>
              <td>{fmtTime(f.captured_at)}</td>
              <td>{f.method}</td><td>{f.host}</td><td>{f.path}</td>
              <td>{f.status_code}</td><td>{f.duration_ms}ms</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if error}
      <p class="error">{error}</p>
    {:else}
      <p class="empty">No flows captured</p>
    {/if}
  </section>

  <section>
    <h3>Connection Activity</h3>
    {#if node.last_seen}
      <ul class="activity">
        <li>Last seen: {fmtDate(node.last_seen)}</li>
        <li>Last flow: {flows.length ? fmtDate(flows[0].captured_at) : 'N/A'}</li>
        <li>Total flows: {flows.length}</li>
        <li>Node status: {node.status || 'unknown'}</li>
      </ul>
    {:else}
      <p class="empty">No activity data</p>
    {/if}
  </section>
</div>

<style>
  .overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 99;
  }
  .panel {
    position: fixed; top: 0; right: 0; bottom: 0; width: 520px;
    background: #0d1117; border-left: 1px solid #30363d; z-index: 100;
    overflow-y: auto; padding: 24px;
  }
  .close {
    position: absolute; top: 12px; right: 16px;
    background: none; border: none; color: #8b949e; font-size: 24px;
    cursor: pointer;
  }
  .close:hover { color: #c9d1d9; }
  h2 { margin: 0 0 20px; color: #c9d1d9; font-size: 20px; }
  section { margin-bottom: 24px; }
  section h3 { font-size: 12px; color: #8b949e; text-transform: uppercase; margin: 0 0 12px; }
  dl { margin: 0; }
  dt { font-size: 11px; color: #8b949e; margin-top: 8px; }
  dd { margin: 2px 0 0; font-size: 14px; color: #c9d1d9; }
  .status-up { color: #3fb950; }
  .status-down { color: #f85149; }
  .token-row {
    display: flex; align-items: center; gap: 8px;
    background: #161b22; padding: 8px 12px; border-radius: 6px;
    border: 1px solid #30363d;
  }
  .token-row code { font-size: 13px; color: #c9d1d9; flex: 1; }
  .copy-btn {
    background: #21262d; border: 1px solid #30363d; color: #c9d1d9;
    padding: 4px 12px; border-radius: 6px; font-size: 12px; cursor: pointer;
  }
  .copy-btn:hover { background: #30363d; }
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid #21262d; }
  th { color: #8b949e; font-weight: 600; }
  .error { color: #f85149; font-size: 13px; }
  .empty { color: #8b949e; font-size: 13px; }
  .activity { list-style: none; padding: 0; margin: 0; }
  .activity li {
    padding: 6px 0; border-bottom: 1px solid #21262d;
    font-size: 13px; color: #c9d1d9;
  }
</style>
