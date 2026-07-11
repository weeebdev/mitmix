<script lang="ts">
  import { onMount } from 'svelte'
  import { listTokens, generateToken, deleteToken } from '../api'
  import type { Token } from '../api'

  let tokens = $state<Token[]>([])
  let error = $state('')
  let label = $state('')

  async function load() {
    try { tokens = await listTokens() }
    catch (e: any) { error = e.message }
  }

  async function submit() {
    if (!label.trim()) return
    try {
      await generateToken(label.trim())
      label = ''
      await load()
    } catch (e: any) { alert('Failed: ' + e.message) }
  }

  async function doDelete(id: string, lbl: string) {
    if (!confirm(`Delete token "${lbl || id}"? Agents using this token will lose access.`)) return
    try {
      await deleteToken(id)
      await load()
    } catch (e: any) { alert('Failed: ' + e.message) }
  }

  function copy(tok: string) {
    navigator.clipboard.writeText(tok)
  }

  onMount(load)
</script>

<table>
  <thead><tr><th>Label</th><th>Token</th><th></th></tr></thead>
  <tbody>
    {#if tokens.length}
      {#each tokens as t}
        <tr>
          <td>{t.label || '-'}</td>
          <td><code>{t.token.slice(0, 12)}…</code></td>
          <td class="actions">
            <button class="copy" onclick={() => copy(t.token)} title="Copy token">Copy</button>
            <button class="delete" onclick={() => doDelete(t.id, t.label)} title="Revoke token">Delete</button>
          </td>
        </tr>
      {/each}
    {:else}
      <tr><td colspan="3" class="empty">No tokens defined{error ? ': ' + error : ''}</td></tr>
    {/if}
  </tbody>
</table>

<form onsubmit={(e) => { e.preventDefault(); submit() }}>
  <h3>Generate Token</h3>
  <label>Label <input bind:value={label} placeholder="e.g. production-agent" /></label>
  <button type="submit">Generate</button>
</form>

<style>
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #21262d; font-size: 13px; }
  th { color: #8b949e; font-weight: 600; }
  td code { font-size: 12px; color: #8b949e; }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
  .actions { white-space: nowrap; text-align: right; }
  .actions button { padding: 3px 10px; border: 1px solid #30363d; border-radius: 4px; cursor: pointer; font-size: 11px; margin-left: 6px; }
  button.copy { background: #21262d; color: #c9d1d9; }
  button.copy:hover { background: #30363d; }
  button.delete { background: none; color: #f85149; }
  button.delete:hover { background: #3d1414; }
  form { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; }
  form h3 { margin-bottom: 12px; font-size: 14px; }
  label { display: block; font-size: 12px; color: #8b949e; margin: 8px 0 4px; }
  input { width: 100%; padding: 6px 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 13px; }
  button[type="submit"] { padding: 6px 16px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; margin-top: 8px; }
  button[type="submit"]:hover { background: #2ea043; }
</style>
