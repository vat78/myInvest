<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAppInfo } from '../wailsjs/go/main/App';

  let name = $state('myInvest');
  let ready = $state(false);
  let error = $state('');

  onMount(() => {
    GetAppInfo().then((info) => {
      name = info.name;
      ready = true;
    }).catch((reason: unknown) => {
      error = `Не удалось запустить приложение: ${String(reason)}`;
    });
  });
</script>

<main>
  <span class="eyebrow">Личный учёт инвестиций</span>
  <h1>{name}</h1>
  <p>Ваши инвестиции. Ваши данные.</p>
  <div class="status" role="status" aria-live="polite">
    {#if error}
      <span class="error">{error}</span>
    {:else if ready}
      <span class="dot"></span> Приложение готово к работе
    {:else}
      Подготовка приложения…
    {/if}
  </div>
</main>

<style>
  :global(body) { margin: 0; background: #f5f7fa; color: #172a3a; font-family: system-ui, sans-serif; }
  :global(*) { box-sizing: border-box; }
  main { max-width: 760px; margin: 0 auto; padding: 100px 40px; }
  .eyebrow { color: #506477; font-size: 14px; letter-spacing: .05em; }
  h1 { margin: 16px 0; font-size: 56px; font-weight: 650; letter-spacing: -.04em; }
  p { color: #506477; font-size: 20px; }
  .status { margin-top: 48px; padding: 20px 24px; border: 1px solid #dce3e9; border-radius: 12px; background: white; line-height: 1.6; }
  .dot { display: inline-block; height: 9px; width: 9px; margin-right: 8px; border-radius: 50%; background: #207653; }
  .error { color: #a12e2e; overflow-wrap: anywhere; }
</style>
