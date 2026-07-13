<script lang="ts">
  import { onMount } from 'svelte'
  import { isAuthed, logout } from './api'
  import Login from './lib/Login.svelte'
  import Nodes from './lib/Nodes.svelte'
  import Rules from './lib/Rules.svelte'
  import Flows from './lib/Flows.svelte'
  import Tokens from './lib/Tokens.svelte'
  import Queries from './lib/Queries.svelte'
  import Stats from './lib/Stats.svelte'

  let authed = $state(false)
  let tab = $state('nodes')
  let flowFilters = $state<{ host?: string; method?: string; status?: string; app?: string; source?: string }>({})

  onMount(() => { authed = isAuthed() })

  function doLogout() { logout(); authed = false }

  function onNavigate(nav: { tab: string; params?: Record<string, string> }) {
    tab = nav.tab
    if (nav.params) flowFilters = nav.params
  }
</script>

{#if !authed}
  <Login />
{:else}
  <nav>
    <h1>mitmix</h1>
    <button class={tab === 'nodes' ? 'active' : ''} onclick={() => tab = 'nodes'}>Nodes</button>
    <button class={tab === 'rules' ? 'active' : ''} onclick={() => tab = 'rules'}>Rules</button>
    <button class={tab === 'flows' ? 'active' : ''} onclick={() => tab = 'flows'}>Flows</button>
    <button class={tab === 'stats' ? 'active' : ''} onclick={() => tab = 'stats'}>Stats</button>
    <button class={tab === 'queries' ? 'active' : ''} onclick={() => tab = 'queries'}>Queries</button>
    <button class={tab === 'tokens' ? 'active' : ''} onclick={() => tab = 'tokens'}>Tokens</button>
    <span class="spacer"></span>
    <a href="/api/mitm/ca-cert" download class="cert-link">Download CA</a>
    <button class="logout" onclick={doLogout}>Logout</button>
  </nav>

  <main>
    {#if tab === 'nodes'}
      <Nodes />
    {:else if tab === 'rules'}
      <Rules />
    {:else if tab === 'flows'}
      <Flows {flowFilters} {onNavigate} />
    {:else if tab === 'stats'}
      <Stats {onNavigate} />
    {:else if tab === 'queries'}
      <Queries />
    {:else if tab === 'tokens'}
      <Tokens />
    {/if}
  </main>
{/if}

<style>
  :global(*) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; }
  nav { background: #161b22; padding: 12px 24px; display: flex; gap: 12px; align-items: center; border-bottom: 1px solid #30363d; }
  nav h1 { font-size: 18px; color: #58a6ff; }
  nav button { background: none; border: none; color: #8b949e; padding: 4px 12px; border-radius: 4px; cursor: pointer; font-size: 14px; }
  nav button:hover { color: #c9d1d9; background: #21262d; }
  nav button.active { color: #c9d1d9; background: #21262d; }
  nav .spacer { flex: 1; }
  nav .logout { color: #f85149; }
  nav .logout:hover { background: #3d1414; }
  nav .cert-link { color: #58a6ff; text-decoration: none; font-size: 14px; padding: 4px 12px; border-radius: 4px; }
  nav .cert-link:hover { background: #21262d; }
  main { padding: 24px; max-width: 1200px; margin: 0 auto; }
</style>
