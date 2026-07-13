<script lang="ts">
  import { onMount } from 'svelte'
  import { listTokens, generateToken, deleteToken } from '../api'
  import type { Token } from '../api'

  let tokens = $state<Token[]>([])
  let error = $state('')
  let label = $state('')
  let lastToken = $state<Token | null>(null)
  let hubHost = $derived(typeof window !== 'undefined' ? window.location.host : 'hub:8090')

  async function load() {
    try { tokens = await listTokens() }
    catch (e: any) { error = e.message }
  }

  async function submit() {
    if (!label.trim()) return
    try {
      lastToken = await generateToken(label.trim())
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

  function closeGuide() {
    lastToken = null
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

{#if lastToken}
  <div class="guide">
    <h3>Agent Setup</h3>
    <button class="close" onclick={closeGuide}>&times;</button>
    <ol>
      <li>
        <strong>Install agent</strong>
        <pre><code>brew tap weeebdev/mitmix
brew install mitmix-agent</code></pre>
      </li>
      <li>
        <strong>Download &amp; install CA cert</strong>
        <pre><code>mitmix-agent --install-cert</code></pre>
      </li>
      <li>
        <strong>Configure &amp; start</strong>
        <pre><code>mitmix-agent --hub ws://{hubHost} --token {lastToken.token}</code></pre>
      </li>
    </ol>
    <p class="hint">Configure your browser/device to use <code>http://agent-ip:8082</code> as HTTP proxy.</p>
  </div>
{/if}

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
  .guide { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; margin-top: 16px; position: relative; }
  .guide h3 { font-size: 14px; margin-bottom: 8px; }
  .guide ol { margin: 0; padding-left: 20px; font-size: 13px; }
  .guide li { margin: 12px 0; }
  .guide pre { background: #0d1117; padding: 8px 12px; border-radius: 4px; margin: 4px 0; overflow-x: auto; }
  .guide code { font-size: 12px; color: #8b949e; }
  .guide .hint { color: #8b949e; font-size: 12px; margin-top: 8px; }
  .guide .close { position: absolute; top: 8px; right: 12px; background: none; border: none; color: #8b949e; font-size: 20px; cursor: pointer; padding: 0; line-height: 1; }
  .guide .close:hover { color: #c9d1d9; }
</style>
