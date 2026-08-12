<script lang="ts">
  import { onMount } from "svelte";
  import { SystemSettingsService } from "@bindings/services/index.js";
  import { m } from "../paraglide/messages.js";
  import { getLocaleVersion, setLanguagePreference } from "$lib/locale.svelte.js";
  import { locales } from "../paraglide/runtime.js";

  export type ThemeMode = "dark" | "light" | "system";
  export type WindowMode = "window" | "fullscreen";

  let {
    showModal = $bindable(false),
    theme = $bindable("system"),
    onchangetheme,
  } = $props<{
    showModal: boolean;
    theme: ThemeMode;
    onchangetheme: (newTheme: ThemeMode) => void;
  }>();

  let dataDir = $state("Loading storage path...");
  let isOpeningFolder = $state(false);
  let openError = $state<string | null>(null);
  let selectedLanguage = $state("system");

  let windowMode = $state<WindowMode>("window");

  async function loadDataDir() {
    try {
      const dir = await SystemSettingsService.GetDataDir();
      if (dir) {
        dataDir = dir;
      }
    } catch (err) {
      console.error("Failed to get data directory path:", err);
      dataDir = "Unable to resolve storage path";
    }
  }

  async function loadLanguage() {
    try {
      const lang = await SystemSettingsService.GetLanguage();
      selectedLanguage = lang || "system";
    } catch (err) {
      console.error("Failed to get language setting:", err);
    }
  }

  async function loadWindowSettings() {
    try {
      const mode = await SystemSettingsService.GetWindowMode();
      if (mode === "window" || mode === "fullscreen") {
        windowMode = mode as WindowMode;
      }
    } catch (err) {
      console.error("Failed to load window settings:", err);
    }
  }

  async function handleSelectLanguage(lang: string) {
    const previousLanguage = selectedLanguage;
    selectedLanguage = lang;
    try {
      await setLanguagePreference(lang);
    } catch (err) {
      console.warn("Failed to set language preference:", err);
      selectedLanguage = previousLanguage;
    }
  }

  async function handleSelectWindowMode(mode: WindowMode) {
    windowMode = mode;
    try {
      await SystemSettingsService.SetWindowMode(mode);
    } catch (err) {
      console.error("Failed to set window mode:", err);
    }
  }

  async function handleOpenFolder() {
    isOpeningFolder = true;
    openError = null;
    try {
      await SystemSettingsService.OpenDataDir();
    } catch (err: any) {
      console.error("Failed to open data directory:", err);
      openError = err?.message || "Could not open system file manager";
    } finally {
      isOpeningFolder = false;
    }
  }

  function handleSelectTheme(selectedTheme: ThemeMode) {
    theme = selectedTheme;
    onchangetheme(selectedTheme);
  }

  $effect(() => {
    if (showModal) {
      loadDataDir();
      loadLanguage();
      loadWindowSettings();
    }
  });

  onMount(() => {
    loadDataDir();
    loadLanguage();
    loadWindowSettings();
  });
</script>

{#if showModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-4 backdrop-blur-sm animate-fadeIn"
    onclick={() => (showModal = false)}
    onkeydown={(e) => e.key === "Escape" && (showModal = false)}
    role="presentation"
  >
    <div
      class="w-full max-w-lg rounded-2xl border border-slate-700 bg-slate-900 p-6 shadow-2xl overflow-hidden text-slate-100 dark-modal-box max-h-[90vh] overflow-y-auto"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="settings-title"
      tabindex="-1"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-slate-800 pb-4 mb-5">
        <div class="flex items-center gap-3">
          <div
            class="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/20 text-sky-400 border border-sky-500/30"
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="h-5 w-5"
            >
              <circle cx="12" cy="12" r="3"></circle>
              <path
                d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
              ></path>
            </svg>
          </div>
          <div>
            <h2 id="settings-title" class="m-0 text-base font-semibold text-white tracking-tight">
              {(getLocaleVersion(), m.settings_title())}
            </h2>
            <p class="m-0 text-xs text-slate-400">{m.settings_subtitle()}</p>
          </div>
        </div>
        <button
          onclick={() => (showModal = false)}
          class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white transition-colors cursor-pointer"
          aria-label="Close settings"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-5 w-5"
          >
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>

      <!-- Content sections -->
      <div class="space-y-5">
        <!-- Appearance / Theme Section -->
        <div>
          <span
            class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2.5"
          >
            {m.settings_section_appearance()}
          </span>
          <div class="grid grid-cols-3 gap-2.5">
            <!-- System Theme -->
            <button
              type="button"
              onclick={() => handleSelectTheme("system")}
              class={`flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all cursor-pointer ${
                theme === "system"
                  ? "border-sky-500 bg-sky-500/10 text-white ring-1 ring-sky-500/50 shadow-md"
                  : "border-slate-800 bg-slate-950/60 text-slate-400 hover:border-slate-700 hover:text-slate-200"
              }`}
            >
              <div class="text-xl mb-1">💻</div>
              <div class="text-xs font-semibold">
                {m.settings_theme_system()}
              </div>
              <div class="text-[10px] text-slate-400 mt-0.5">
                {m.settings_theme_system_sub()}
              </div>
            </button>

            <!-- Dark Mode -->
            <button
              type="button"
              onclick={() => handleSelectTheme("dark")}
              class={`flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all cursor-pointer ${
                theme === "dark"
                  ? "border-sky-500 bg-sky-500/10 text-white ring-1 ring-sky-500/50 shadow-md"
                  : "border-slate-800 bg-slate-950/60 text-slate-400 hover:border-slate-700 hover:text-slate-200"
              }`}
            >
              <div class="text-xl mb-1">🌙</div>
              <div class="text-xs font-semibold">{m.settings_theme_dark()}</div>
              <div class="text-[10px] text-slate-400 mt-0.5">
                {m.settings_theme_dark_sub()}
              </div>
            </button>

            <!-- Light Mode -->
            <button
              type="button"
              onclick={() => handleSelectTheme("light")}
              class={`flex flex-col items-center justify-center rounded-xl border p-3 text-center transition-all cursor-pointer ${
                theme === "light"
                  ? "border-sky-500 bg-sky-500/10 text-slate-900 ring-1 ring-sky-500/50 shadow-md"
                  : "border-slate-800 bg-slate-950/60 text-slate-400 hover:border-slate-700 hover:text-slate-200"
              }`}
            >
              <div class="text-xl mb-1">☀️</div>
              <div class="text-xs font-semibold">
                {m.settings_theme_light()}
              </div>
              <div class="text-[10px] text-slate-400 mt-0.5">
                {m.settings_theme_light_sub()}
              </div>
            </button>
          </div>
        </div>

        <!-- Display & Window Mode Section -->
        <div>
          <span
            class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >
            {m.settings_section_display()}
          </span>
          <div
            class="flex items-center justify-between gap-3 rounded-xl border border-slate-800 bg-slate-950/80 px-4 py-3"
          >
            <div class="flex flex-col">
              <span class="text-xs font-semibold text-slate-200">
                {windowMode === "fullscreen"
                  ? m.settings_window_mode_fullscreen()
                  : m.settings_window_mode_window()}
              </span>
            </div>

            <button
              type="button"
              role="switch"
              aria-label="Toggle fullscreen display mode"
              aria-checked={windowMode === "fullscreen"}
              onclick={() =>
                handleSelectWindowMode(windowMode === "fullscreen" ? "window" : "fullscreen")}
              class={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-sky-500 focus:ring-offset-2 focus:ring-offset-slate-900 ${
                windowMode === "fullscreen" ? "bg-sky-500" : "bg-slate-800"
              }`}
            >
              <span
                class={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                  windowMode === "fullscreen" ? "translate-x-5" : "translate-x-0"
                }`}
              ></span>
            </button>
          </div>
        </div>

        <!-- Language Section -->
        <div>
          <span
            class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >
            {m.settings_section_language()}
          </span>
          <select
            id="language-select"
            value={selectedLanguage}
            onchange={(e) => handleSelectLanguage((e.target as HTMLSelectElement).value)}
            class="w-full rounded-xl border border-slate-800 bg-slate-950/80 px-3.5 py-2.5 text-sm text-slate-200 focus:border-sky-500 focus:outline-none focus:ring-1 focus:ring-sky-500/50 transition-all cursor-pointer"
          >
            <option value="system">🌐 {m.settings_language_system()}</option>
            {#each locales as locale}
              <option value={locale}>🗣 {locale.toUpperCase()}</option>
            {/each}
          </select>
        </div>

        <!-- Ultra-compact Data Storage Directory Section -->
        <div>
          <span
            class="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2"
          >
            {m.settings_section_storage()}
          </span>

          <div
            class="flex items-center justify-between gap-3 rounded-xl border border-slate-800 bg-slate-950/80 px-3.5 py-2.5"
          >
            <div class="flex items-center gap-2.5 min-w-0 overflow-hidden">
              <span class="text-slate-400 text-sm shrink-0">📁</span>
              <span class="font-mono text-xs text-slate-300 truncate select-all">{dataDir}</span>
            </div>

            <button
              type="button"
              onclick={handleOpenFolder}
              disabled={isOpeningFolder}
              class="btn btn-secondary btn-sm shrink-0 text-xs py-1 px-3 gap-1.5 cursor-pointer"
              title="Open Storage Folder"
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                class="h-3.5 w-3.5"
              >
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                <polyline points="15 3 21 3 21 9"></polyline>
                <line x1="10" y1="14" x2="21" y2="3"></line>
              </svg>
              <span
                >{isOpeningFolder ? m.settings_storage_opening() : m.settings_storage_open()}</span
              >
            </button>
          </div>

          {#if openError}
            <div
              class="mt-2 text-xs text-rose-400 bg-rose-500/10 border border-rose-500/20 p-2 rounded-lg"
            >
              ⚠️ {openError}
            </div>
          {/if}
        </div>
      </div>

      <!-- Footer -->
      <div class="mt-6 flex justify-end border-t border-slate-800 pt-4">
        <button
          type="button"
          onclick={() => (showModal = false)}
          class="btn btn-primary px-5 py-2 text-xs cursor-pointer"
        >
          {m.settings_done()}
        </button>
      </div>
    </div>
  </div>
{/if}
