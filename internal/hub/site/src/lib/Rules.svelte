<script lang="ts">
  import { onMount } from 'svelte'
  import { listRules, createRule } from '../api'
  import type { Rule } from '../api'

  let rules = $state<Rule[]>([])
  let error = $state('')

  let action = $state('record')
  let node = $state('*')
  let priority = $state(0)
  let match = $state('{"host": "*"}')
  let spec = $state('{}')
  let enabled = $state(true)

  async function load() {
    try { rules = await listRules() }
    catch (e: any) { error = e.message }
  }

  async function submit() {
    try {
      await createRule({ node: node || '*', priority, action, match: match || '{}', spec: spec || '{}', enabled })
      action = 'record'; node = '*'; priority = 0; match = '{"host": "*"}'; spec = '{}'; enabled = true
      await load()
    } catch (e: any) { alert('Failed: ' + e.message) }
  }

  onMount(load)
</script>

<table>
  <thead><tr><th>Priority</th><th>Node</th><th>Action</th><th>Match</th><th>Enabled</th></tr></thead>
  <tbody>
    {#if rules.length}
      {#each rules as r}
        <tr>
          <td>{r.priority}</td><td>{r.node}</td>
          <td><span class="badge">{r.action}</span></td>
          <td><code>{r.match}</code></td>
          <td>{r.enabled ? '✅' : '❌'}</td>
        </tr>
      {/each}
    {:else}
      <tr><td colspan="5" class="empty">No rules defined{error ? ': ' + error : ''}</td></tr>
    {/if}
  </tbody>
</table>

<form onsubmit={(e) => { e.preventDefault(); submit() }}>
  <h3>Create Rule</h3>
  <label>Action <select bind:value={action}><option>record</option><option>intercept</option><option>drop</option><option>redirect</option><option>modify_headers</option><option>modify_body</option></select></label>
  <label>Node <input bind:value={node} /></label>
  <label>Priority <input type="number" bind:value={priority} /></label>
  <label>Match (JSON) <textarea bind:value={match} rows="2"></textarea></label>
  <label>Spec (JSON) <textarea bind:value={spec} rows="2"></textarea></label>
  <label><input type="checkbox" bind:checked={enabled} /> Enabled</label>
  <button type="submit">Create Rule</button>
</form>

<style>
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 11px; font-weight: 600; background: #1b3624; color: #3fb950; }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
  form { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; }
  form h3 { margin-bottom: 12px; font-size: 14px; }
  label { display: block; font-size: 12px; color: #8b949e; margin: 8px 0 4px; }
  input, select, textarea { width: 100%; padding: 6px 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 13px; }
  textarea { font-family: monospace; }
  button { padding: 6px 16px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; margin-top: 8px; }
  button[type="submit"]:hover { background: #2ea043; }
</style>
