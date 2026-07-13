<script lang="ts">
  import { onMount } from 'svelte'
  import { listRules, createRule, updateRule as updateRuleApi, deleteRule as deleteRuleApi, reorderRules } from '../api'
  import type { Rule } from '../api'

  let rules = $state<Rule[]>([])
  let error = $state('')
  let editingId = $state<string | null>(null)

  let action = $state('record')
  let node = $state('*')
  let priority = $state(0)
  let match = $state('{"host": "*"}')
  let spec = $state('{}')
  let enabled = $state(true)

  let editAction = $state('')
  let editPriority = $state(0)
  let editMatch = $state('')
  let editSpec = $state('')

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

  function fmtObj(v: any): string {
    if (typeof v === 'string') return v
    try { return JSON.stringify(v) } catch { return String(v) }
  }

  function startEdit(r: Rule) {
    editingId = r.id
    editAction = r.action
    editPriority = r.priority
    editMatch = fmtObj(r.match)
    editSpec = fmtObj(r.spec)
  }

  function cancelEdit() {
    editingId = null
  }

  async function saveEdit(r: Rule) {
    try {
      await updateRuleApi(r.id, { action: editAction, priority: editPriority, match: editMatch, spec: editSpec })
      editingId = null
      await load()
    } catch (e: any) { alert('Failed: ' + e.message) }
  }

  async function del(r: Rule) {
    if (!confirm('Delete rule "' + r.id.slice(0, 8) + '"?')) return
    try {
      await deleteRuleApi(r.id)
      await load()
    } catch (e: any) { alert('Failed: ' + e.message) }
  }

  function moveItem(fromId: string, toId: string) {
    const from = rules.find(r => r.id === fromId)
    const to = rules.find(r => r.id === toId)
    if (!from || !to) return
    const copy = [...rules]
    const fi = copy.indexOf(from)
    const ti = copy.indexOf(to)
    copy.splice(fi, 1)
    copy.splice(ti, 0, from)
    rules = copy
    reorderRules(rules.map(r => r.id)).catch((e: any) => { alert('Reorder failed: ' + e.message); load() })
  }

  onMount(load)
</script>

<table>
  <thead><tr><th></th><th>Priority</th><th>Node</th><th>Action</th><th>Match</th><th>Enabled</th><th></th></tr></thead>
  <tbody>
    {#if rules.length}
      {#each rules as r}
        <tr
          draggable="true"
          ondragstart={(e) => { e.dataTransfer?.setData('text/plain', r.id) }}
          ondragover={(e) => e.preventDefault()}
          ondrop={(e) => { e.preventDefault(); const id = e.dataTransfer?.getData('text/plain'); if (id) moveItem(id, r.id) }}
        >
          <td class="drag-handle">⠿</td>
          {#if editingId === r.id}
            <td><input type="number" class="inline" bind:value={editPriority} style="width:60px" /></td>
            <td>{r.node}</td>
            <td>
              <select bind:value={editAction}>
                <option>record</option><option>intercept</option><option>drop</option><option>redirect</option>
                <option>modify_headers</option><option>modify_body</option>
                <option>copy_request</option><option>replicate</option><option>rewrite</option><option>write_rule</option>
              </select>
            </td>
            <td><input type="text" class="inline" bind:value={editMatch} style="width:120px;font-family:monospace" /></td>
            <td>{r.enabled ? '✅' : '❌'}</td>
            <td class="actions">
              <button class="small" onclick={() => saveEdit(r)}>Save</button>
              <button class="small cancel" onclick={cancelEdit}>Cancel</button>
            </td>
          {:else}
            <td>{r.priority}</td><td>{r.node}</td>
            <td><span class="badge">{r.action}</span></td>
            <td><code>{fmtObj(r.match)}</code></td>
            <td>{r.enabled ? '✅' : '❌'}</td>
            <td class="actions">
              <button class="small" onclick={() => startEdit(r)}>Edit</button>
              <button class="small danger" onclick={() => del(r)}>Delete</button>
            </td>
          {/if}
        </tr>
      {/each}
    {:else}
      <tr><td colspan="7" class="empty">No rules defined{error ? ': ' + error : ''}</td></tr>
    {/if}
  </tbody>
</table>

<form onsubmit={(e) => { e.preventDefault(); submit() }}>
  <h3>Create Rule</h3>
  <label>Action
    <select bind:value={action}>
      <option>record</option><option>intercept</option><option>drop</option><option>redirect</option>
      <option>modify_headers</option><option>modify_body</option>
      <option>copy_request</option><option>replicate</option><option>rewrite</option><option>write_rule</option>
    </select>
  </label>
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
  tr { cursor: grab; }
  tr:hover { background: #161b22; }
  .drag-handle { cursor: grab; color: #484f58; font-size: 16px; user-select: none; width: 24px; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 11px; font-weight: 600; background: #1b3624; color: #3fb950; }
  .empty { color: #8b949e; text-align: center; padding: 24px; }
  .actions { white-space: nowrap; }
  .small { padding: 2px 8px; font-size: 11px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 4px; cursor: pointer; margin-right: 4px; }
  .small:hover { background: #30363d; }
  .small.danger { color: #f85149; border-color: #f85149; }
  .small.danger:hover { background: #3d1414; }
  .small.cancel { color: #8b949e; }
  .inline { background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; padding: 4px 6px; font-size: 13px; }
  form { background: #161b22; padding: 16px; border-radius: 6px; border: 1px solid #30363d; }
  form h3 { margin-bottom: 12px; font-size: 14px; }
  label { display: block; font-size: 12px; color: #8b949e; margin: 8px 0 4px; }
  input, select, textarea { width: 100%; padding: 6px 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 13px; }
  textarea { font-family: monospace; }
  button[type="submit"] { padding: 6px 16px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; margin-top: 8px; }
  button[type="submit"]:hover { background: #2ea043; }
</style>
