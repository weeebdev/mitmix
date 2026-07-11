<script lang="ts">
  import { onMount } from 'svelte'
  import { listQueries, createQuery, deleteQuery, runQuery, fmtTime } from '../api'
  import type { Flow, Query } from '../api'
  import FlowDetail from './FlowDetail.svelte'

  let queries = $state<Query[]>([])
  let results = $state<Flow[]>([])
  let selected = $state<Flow | null>(null)
  let name = $state('')
  let filter = $state('')
  let error = $state('')

  onMount(load)

  async function load() {
    try { queries = await listQueries() }
    catch (e: any) { error = e.message }
  }

  async function add() {
    if (!name) return
    await createQuery(name, filter || '{}')
    name = ''; filter = ''
    load()
  }

  async function del(id: string) {
    await deleteQuery(id)
    load()
    if (results.length) results = []
  }

  async function run(id: string) {
    try { results = await runQuery(id) }
    catch (e: any) { error = e.message }
  }
</script>

<div style="display:flex;gap:20px;align-items:flex-start">
  <div style="flex:1">
    <h3>Saved Queries</h3>

    <div class="form">
      <input type="text" placeholder="Query name" bind:value={name} />
      <input type="text" placeholder='Filter JSON e.g. host:example method:GET' bind:value={filter} />
      <button onclick={add}>Save</button>
    </div>

    {#if error}
      <p class="err">{error}</p>
    {/if}

    <table>
      <thead><tr><th>Name</th><th>Filter</th><th></th></tr></thead>
      <tbody>
        {#each queries as q}
          <tr>
            <td>{q.name}</td>
            <td style="font-family:monospace;font-size:12px">{q.filter}</td>
            <td class="actions">
              <button onclick={() => run(q.id)}>Run</button>
              <button class="del" onclick={() => del(q.id)}>Del</button>
            </td>
          </tr>
        {:else}
          <tr><td colspan="3" class="empty">No saved queries</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <div style="flex:1">
    {#if results.length}
      <h3>Results</h3>
      <table>
        <thead><tr><th>Time</th><th>Method</th><th>Host</th><th>Path</th><th>Status</th></tr></thead>
        <tbody>
          {#each results as f}
            <tr class:selected={selected?.id === f.id} onclick={() => selected = f}>
              <td>{fmtTime(f.captured_at)}</td>
              <td>{f.method}</td><td>{f.host}</td><td>{f.path}</td><td>{f.status_code}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}

    {#if selected}
      <FlowDetail flow={selected} onclose={() => selected = null} />
    {/if}
  </div>
</div>

<style>
  .form { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
  .form input { flex: 1; padding: 6px 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 13px; min-width: 120px; }
  .form button { padding: 4px 12px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 4px; cursor: pointer; font-size: 13px; }
  .form button:hover { background: #30363d; }
  .err { color: #f85149; font-size: 13px; margin-bottom: 8px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { text-align: left; padding: 6px 10px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  tr { cursor: pointer; }
  tr:hover { background: #161b22; }
  tr.selected { background: #1c2128; }
  .empty { color: #8b949e; text-align: center; padding: 16px; }
  .actions { text-align: right; white-space: nowrap; }
  .actions button { background: none; border: 1px solid #30363d; color: #c9d1d9; padding: 2px 8px; border-radius: 3px; cursor: pointer; font-size: 12px; margin-left: 4px; }
  .actions .del { color: #f85149; }
  h3 { font-size: 14px; color: #8b949e; text-transform: uppercase; margin-bottom: 12px; }
</style>
