<script lang="ts">
  import { login } from '../api'

  let identity = $state('admin@mitm.local')
  let password = $state('mitmadmin123')
  let error = $state('')
  let loading = $state(false)

  async function submit(e: Event) {
    e.preventDefault()
    loading = true; error = ''
    try {
      await login(identity, password)
      window.location.reload()
    } catch (e: any) {
      error = e.message || 'Login failed'
    } finally { loading = false }
  }
</script>

<div class="wrap">
  <form onsubmit={submit}>
    <h1>mitm-decentralized</h1>
    <p class="sub">Sign in to your account</p>
    {#if error}<div class="error">{error}</div>{/if}
    <label>Email <input type="text" bind:value={identity} placeholder="admin@mitm.local" /></label>
    <label>Password <input type="password" bind:value={password} /></label>
    <button type="submit" disabled={loading}>{loading ? 'Signing in…' : 'Sign In'}</button>
  </form>
</div>

<style>
  .wrap { display: flex; align-items: center; justify-content: center; min-height: 100vh; }
  form { background: #161b22; padding: 32px; border-radius: 8px; border: 1px solid #30363d; width: 360px; }
  h1 { font-size: 22px; color: #58a6ff; margin-bottom: 4px; text-align: center; }
  .sub { font-size: 13px; color: #8b949e; text-align: center; margin-bottom: 24px; }
  .error { background: #3d1414; border: 1px solid #f85149; color: #f85149; padding: 8px; border-radius: 4px; font-size: 13px; margin-bottom: 12px; text-align: center; }
  label { display: block; font-size: 12px; color: #8b949e; margin: 12px 0 4px; }
  input { width: 100%; padding: 8px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 14px; box-sizing: border-box; }
  button { width: 100%; padding: 10px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; margin-top: 20px; font-weight: 600; }
  button:hover { background: #2ea043; }
  button:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
