<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { listFlows } from '../api'
  import type { Flow } from '../api'
  import FlowDetail from './FlowDetail.svelte'

  let flows = $state<Flow[]>([])
  let selected = $state<Flow | null>(null)
  let error = $state('')
  let interval: number

  async function load() {
    try { flows = await listFlows() }
    catch (e: any) { error = e.message }
  }

  onMount(() => { load(); interval = setInterval(load, 5000) })
  onDestroy(() => clearInterval(interval))
</script>

<div class="card"><h3>Captured Flows</h3><div class="val">{flows.length}</div></div>

<div class="flows-layout">
  <div class="flows-table">
    <table>
      <thead><tr><th>Time</th><th>Method</th><th>Host</th><th>Path</th><th>Status</th><th>Duration</th></tr></thead>
      <tbody>
        {#if flows.length}
          {#each flows as f}
            <tr class:selected={selected?.id === f.id} onclick={() => selected = f}>
              <td>{f.captured_at ? new Date(f.captured_at).toLocaleTimeString() : f.id?.slice(0, 8)}</td>
              <td>{f.method}</td><td>{f.host}</td><td>{f.path}</td>
              <td>{f.status_code}</td><td>{f.duration_ms}ms</td>
            </tr>
          {/each}
        {:else}
          <tr><td colspan="6" class="empty">No flows captured{error ? ': ' + error : ''}</td></tr>
        {/if}
      </tbody>
    </table>
  </div>

  {#if selected}
    <div class="detail-panel">
      <FlowDetail flow={selected} onclose={() => selected = null} />
    </div>
  {/if}
</div>

<style>
  .card { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; margin-bottom: 16px; display: inline-block; }
  .card h3 { font-size: 12px; color: #8b949e; text-transform: uppercase; }
  .card .val { font-size: 28px; font-weight: 700; }
  .flows-layout { display: flex; gap: 16px; align-items: flex-start; }
  .flows-table { flex: 1; min-width: 0; }
  .detail-panel { flex-shrink: 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  tr { cursor: pointer; }
  tr:hover { background: #161b22; }
  tr.selected { background: #1c2128; }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
</style>
