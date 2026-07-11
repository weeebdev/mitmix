<script lang="ts">
  import { onMount } from 'svelte'
  import { listNodes } from '../api'
  import type { Node } from '../api'

  let nodes = $state<Node[]>([])
  let error = $state('')

  onMount(async () => {
    try { nodes = await listNodes() }
    catch (e: any) { error = e.message }
  })
</script>

<div class="card"><h3>Connected Nodes</h3><div class="val">{nodes.length}</div></div>

<table>
  <thead><tr><th>Name</th><th>Token</th><th>Fingerprint</th><th>Status</th><th>Version</th></tr></thead>
  <tbody>
    {#if nodes.length}
      {#each nodes as n}
        <tr>
          <td>{n.name || n.id}</td>
          <td>{n.token?.slice(0, 8)}…</td>
          <td>{n.fingerprint || '-'}</td>
          <td class="status-{n.status}">{n.status || 'down'}</td>
          <td>{n.version || '-'}</td>
        </tr>
      {/each}
    {:else}
      <tr><td colspan="5" class="empty">No nodes connected{error ? ': ' + error : ''}</td></tr>
    {/if}
  </tbody>
</table>

<style>
  .card { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; margin-bottom: 16px; display: inline-block; }
  .card h3 { font-size: 12px; color: #8b949e; text-transform: uppercase; }
  .card .val { font-size: 28px; font-weight: 700; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  .status-up { color: #3fb950; }
  .status-down { color: #f85149; }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
</style>
