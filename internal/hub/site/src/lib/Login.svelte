<script lang="ts">
  import { onMount } from 'svelte'
  import { login, fetchAuthMethods, handleOAuthRedirect } from '../api'
  import type { AuthProvider } from '../api'

  let identity = $state('')
  let password = $state('')
  let error = $state('')
  let loading = $state(false)
  let providers = $state<AuthProvider[]>([])
  let pwEnabled = $state(true)

  onMount(async () => {
    if (handleOAuthRedirect()) {
      window.location.reload()
      return
    }
    try {
      const m = await fetchAuthMethods()
      pwEnabled = m.password.enabled
      if (m.oauth2.enabled) providers = m.oauth2.providers
    } catch {}
  })

  function oauthLogin(p: AuthProvider) {
    if (p.codeVerifier) sessionStorage.setItem('pkce_' + p.name, p.codeVerifier)
    window.location.href = p.authUrl
  }

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
    <h1>mitmix</h1>
    <p class="sub">Sign in to your account</p>
    {#if error}<div class="error">{error}</div>{/if}
    {#if pwEnabled}
      <label>Email <input type="text" bind:value={identity} placeholder="email" /></label>
      <label>Password <input type="password" bind:value={password} /></label>
      <button type="submit" disabled={loading}>{loading ? 'Signing in…' : 'Sign In'}</button>
    {/if}
    {#if providers.length}
      {#if pwEnabled}<div class="divider"><span>or</span></div>{/if}
      {#each providers as p}
        <button type="button" class="oauth" onclick={() => oauthLogin(p)}>
          Sign in with {p.displayName}
        </button>
      {/each}
    {/if}
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
  .oauth { background: #21262d; color: #c9d1d9; border: 1px solid #30363d; margin-top: 8px; }
  .oauth:hover { background: #30363d; }
  .divider { display: flex; align-items: center; margin: 20px 0 0; color: #8b949e; font-size: 13px; }
  .divider::before, .divider::after { content: ''; flex: 1; border-top: 1px solid #30363d; }
  .divider span { padding: 0 12px; }
</style>
