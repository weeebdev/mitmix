<script lang="ts">
  import { onMount } from 'svelte'
  import Nodes from './lib/Nodes.svelte'
  import Rules from './lib/Rules.svelte'
  import Flows from './lib/Flows.svelte'

  let tab = $state('nodes')
</script>

<nav>
  <h1>mitm-decentralized</h1>
  <button class={tab === 'nodes' ? 'active' : ''} onclick={() => tab = 'nodes'}>Nodes</button>
  <button class={tab === 'rules' ? 'active' : ''} onclick={() => tab = 'rules'}>Rules</button>
  <button class={tab === 'flows' ? 'active' : ''} onclick={() => tab = 'flows'}>Flows</button>
</nav>

<main>
  {#if tab === 'nodes'}
    <Nodes />
  {:else if tab === 'rules'}
    <Rules />
  {:else if tab === 'flows'}
    <Flows />
  {/if}
</main>

<style>
  :global(*) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; }
  nav { background: #161b22; padding: 12px 24px; display: flex; gap: 12px; align-items: center; border-bottom: 1px solid #30363d; }
  nav h1 { font-size: 18px; color: #58a6ff; }
  nav button { background: none; border: none; color: #8b949e; padding: 4px 12px; border-radius: 4px; cursor: pointer; font-size: 14px; }
  nav button:hover { color: #c9d1d9; background: #21262d; }
  nav button.active { color: #c9d1d9; background: #21262d; }
  main { padding: 24px; max-width: 1200px; margin: 0 auto; }
</style>
