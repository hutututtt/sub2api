<template>
  <div class="night-home" :class="{ 'is-light': lightMode }" data-testid="night-home">
    <div class="ambient ambient-one" aria-hidden="true"></div>
    <div class="ambient ambient-two" aria-hidden="true"></div>

    <header id="top" class="home-header shell">
      <a href="#top" class="brand" :aria-label="siteName">
        <img v-if="siteLogo" :src="siteLogo" alt="" />
        <span v-else class="brand-mark" aria-hidden="true">✦</span>
        <span>{{ siteName }}<small>CONNECT TO POSSIBILITY</small></span>
      </a>
      <nav class="desktop-nav" :aria-label="t('home.night.navigation')">
        <a href="#top" :class="{ active: activeNav === 'home' }" @click="activeNav = 'home'">{{ t('home.night.homeNav') }}</a>
        <a href="#why" :class="{ active: activeNav === 'features' }" @click="activeNav = 'features'">{{ t('home.night.featuresNav') }}</a>
        <a href="#models" :class="{ active: activeNav === 'models' }" @click="activeNav = 'models'">{{ t('home.night.modelsNav') }}</a>
        <a :href="docUrl || '#start'" :target="docUrl ? '_blank' : undefined" :rel="docUrl ? 'noopener noreferrer' : undefined">{{ t('home.docs') }}</a>
        <a href="#about" @click="activeNav = 'about'">{{ t('home.night.aboutNav') }}</a>
      </nav>
      <div class="header-actions">
        <LocaleSwitcher />
        <button class="theme-button" :aria-label="lightMode ? t('home.switchToDark') : t('home.switchToLight')" :title="lightMode ? t('home.switchToDark') : t('home.switchToLight')" @click="toggleLandingTheme">
          <Icon :name="lightMode ? 'moon' : 'sun'" size="sm" />
        </button>
        <router-link :to="entryPath" class="header-login">{{ isAuthenticated ? t('home.dashboard') : t('home.login') }}</router-link>
      </div>
    </header>

    <main>
      <section class="hero shell" aria-labelledby="hero-title">
        <div class="hero-copy">
          <div class="eyebrow"><span aria-hidden="true">✦</span>{{ t('home.night.eyebrow') }}</div>
          <h1 id="hero-title">{{ t('home.night.titleFirst') }}<br />{{ t('home.night.titleLead') }}<em>{{ t('home.night.titleAccent') }}</em>{{ t('home.night.titleEnd') }}</h1>
          <p class="hero-description">{{ t('home.night.description') }}</p>
          <div class="hero-actions">
            <router-link :to="entryPath" class="primary-button">{{ isAuthenticated ? t('home.goToDashboard') : t('home.night.getStarted') }}<Icon name="arrowRight" size="sm" /></router-link>
            <a :href="docUrl || '#start'" :target="docUrl ? '_blank' : undefined" :rel="docUrl ? 'noopener noreferrer' : undefined" class="secondary-button"><Icon name="terminal" size="sm" />{{ docUrl ? t('home.viewDocs') : t('home.night.viewGuide') }}</a>
          </div>
          <div class="hero-notes">
            <span><Icon name="check" size="xs" />{{ t('home.night.noteKey') }}</span>
            <span><Icon name="check" size="xs" />{{ t('home.tags.realtimeBilling') }}</span>
            <span><Icon name="check" size="xs" />{{ t('home.night.noteSwitch') }}</span>
          </div>
        </div>

        <div class="hero-visual" role="img" :aria-label="t('home.night.orbitLabel')">
          <img class="orbit-art" src="/assets/home/ai-orbit.svg" alt="" />
          <div class="orbit-core"><span class="core-spark" aria-hidden="true">✦</span><strong>{{ siteName }}</strong><span>ONE API. ALL AI.</span></div>
          <div class="provider-node node-claude"><ModelIcon model="claude" size="34px" /><div><strong>Claude</strong><small>Anthropic</small></div><span class="node-dot"></span></div>
          <div class="provider-node node-gpt"><ModelIcon model="gpt" size="29px" /><div><strong>GPT</strong><small>OpenAI</small></div><span class="node-dot"></span></div>
          <div class="provider-node node-gemini"><ModelIcon model="gemini" size="34px" /><div><strong>Gemini</strong><small>Google</small></div><span class="node-dot"></span></div>
          <div class="provider-node node-antigravity"><span class="antigravity-mark" aria-hidden="true">Λ</span><div><strong>Antigravity</strong><small>{{ t('home.night.antigravityCaption') }}</small></div></div>
          <div class="orbit-label"><span aria-hidden="true">✦</span>{{ t('home.night.orbitCaption') }}</div>
        </div>
      </section>

      <div class="content-surface">
        <div class="shell">
          <section id="models" class="models-section" aria-labelledby="models-title">
            <div class="section-heading">
              <div><p class="section-kicker">{{ t('home.night.modelsKicker') }}</p><h2 id="models-title">{{ t('home.night.modelsTitle') }}</h2></div>
              <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="text-link">{{ t('home.night.viewModels') }}<Icon name="arrowRight" size="sm" /></router-link>
              <a v-else href="#start" class="text-link">{{ t('home.night.viewGuide') }}<Icon name="arrowRight" size="sm" /></a>
            </div>
            <p class="section-description">{{ t('home.night.modelsDescription') }}</p>
            <div class="model-grid">
              <article class="model-card"><ModelIcon model="claude" size="43px" /><div><h3>Claude</h3><p>Anthropic</p></div></article>
              <article class="model-card"><span class="openai-badge"><ModelIcon model="gpt" size="30px" /></span><div><h3>OpenAI</h3><p>{{ t('home.night.gptFamily') }}</p></div></article>
              <article class="model-card"><ModelIcon model="gemini" size="43px" /><div><h3>Gemini</h3><p>Google</p></div></article>
              <article class="model-card"><span class="antigravity-mark" aria-hidden="true">Λ</span><div><h3>Antigravity</h3><p>{{ t('home.night.antigravityCaption') }}</p></div></article>
            </div>
          </section>

          <section id="why" class="features-section" aria-labelledby="features-title">
            <div class="section-heading"><div><p class="section-kicker">{{ t('home.night.featuresKicker', { site: siteName }) }}</p><h2 id="features-title">{{ t('home.night.featuresTitle') }}</h2></div><p class="heading-aside">{{ t('home.night.featuresSubtitle') }}</p></div>
            <div class="feature-grid">
              <article v-for="(feature, index) in features" :key="feature.key" class="feature-card">
                <div class="feature-top"><span class="feature-icon"><Icon :name="feature.icon" size="lg" /></span><span class="feature-number">0{{ index + 1 }}</span></div>
                <h3>{{ t(`home.night.${feature.key}Title`) }}</h3><p>{{ t(`home.night.${feature.key}Description`) }}</p>
                <div class="feature-footer"><span></span>{{ t(`home.night.${feature.key}Detail`) }}</div>
              </article>
            </div>
          </section>

          <section id="start" class="start-section" aria-labelledby="start-title">
            <div class="guide-card">
              <div class="guide-art" aria-hidden="true"><div class="glass-window window-back"></div><div class="glass-window window-middle"></div><div class="glass-window window-front"><span>&lt;/&gt;</span></div><i class="art-cube cube-one"></i><i class="art-cube cube-two"></i><i class="art-dot"></i></div>
              <p class="section-kicker guide-kicker">HELLO, POSSIBILITY.</p>
              <h2 id="start-title">{{ t('home.night.startTitleFirst') }}{{ t('home.night.startTitleSecond') }}</h2>
              <p class="section-description">{{ t('home.night.startDescription') }}</p>
              <ol class="steps"><li v-for="(step, index) in steps" :key="step"><span class="step-number">0{{ index + 1 }}</span><div><h3>{{ t(`home.night.${step}Title`) }}</h3><p>{{ t(`home.night.${step}Description`) }}</p></div></li></ol>
              <router-link :to="entryPath" class="text-link">{{ isAuthenticated ? t('home.goToDashboard') : t('home.night.createKey') }}<Icon name="arrowRight" size="sm" /></router-link>
            </div>

            <div class="code-section">
              <div class="code-section-heading"><div><h3>{{ t('home.night.codeTitle') }}</h3><p>{{ t('home.night.codeDescription') }}</p></div><div class="language-tabs" :aria-label="t('home.night.exampleLanguage')"><button v-for="language in languages" :key="language.id" :aria-pressed="selectedLanguage === language.id" :class="{ active: selectedLanguage === language.id }" @click="selectedLanguage = language.id">{{ language.name }}</button></div></div>
              <div class="code-settings"><label for="demo-model">{{ t('home.night.exampleModel') }}</label><select id="demo-model" v-model="selectedModel"><option v-for="(model, index) in demoModels" :key="model.name" :value="index">{{ model.name }}</option></select><span>{{ t('home.night.exampleOnly') }}</span></div>
              <div class="terminal-container code-panel">
                <button class="copy-button" :aria-label="t('home.night.copyCode')" @click="copyToClipboard(sampleCode)"><Icon :name="copied ? 'check' : 'copy'" size="xs" />{{ copied ? t('home.night.copied') : t('home.night.copyCode') }}</button>
                <pre><code><span v-for="(line, index) in sampleCode.split('\n')" :key="index" class="code-line"><span class="line-number" aria-hidden="true">{{ index + 1 }}</span><span>{{ line }}</span>
</span></code></pre>
              </div>
              <p class="code-footnote"><Icon name="terminal" size="xs" />{{ t('home.night.codeFootnote') }}</p>
            </div>
          </section>

          <section id="about" class="closing" aria-labelledby="closing-title"><span class="closing-star" aria-hidden="true">✦</span><div><h2 id="closing-title">{{ t('home.night.closingTitle') }}</h2><p>{{ t('home.night.closingDescription') }}</p></div><router-link :to="entryPath" class="primary-button">{{ isAuthenticated ? t('home.goToDashboard') : t('home.night.getStarted') }}<Icon name="arrowRight" size="sm" /></router-link></section>
        </div>
      </div>
    </main>

    <footer class="home-footer"><div class="shell"><div class="footer-top"><a href="#top" class="brand"><img v-if="siteLogo" :src="siteLogo" alt="" /><span v-else class="brand-mark" aria-hidden="true">✦</span><span>{{ siteName }}<small>CONNECT TO POSSIBILITY</small></span></a><nav :aria-label="t('home.night.footerNavigation')"><a href="#top">{{ t('home.night.homeNav') }}</a><a href="#why">{{ t('home.night.featuresNav') }}</a><a href="#models">{{ t('home.night.modelsNav') }}</a><a :href="docUrl || '#start'" :target="docUrl ? '_blank' : undefined" :rel="docUrl ? 'noopener noreferrer' : undefined">{{ t('home.docs') }}</a><a href="#about">{{ t('home.night.aboutNav') }}</a></nav></div><div class="footer-bottom"><span>© {{ new Date().getFullYear() }} · {{ siteName }} {{ t('home.footer.allRightsReserved') }}</span><span>{{ t('home.night.footerLine') }}</span><a href="https://github.com/Wei-Shaw/sub2api" target="_blank" rel="noopener noreferrer">GitHub ↗</a></div></div></footer>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'

const props = defineProps<{
  siteName: string
  siteLogo: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  showModelPlazaEntry: boolean
}>()
const { t } = useI18n()
const appStore = useAppStore()
const { copied, copyToClipboard } = useClipboard()
const lightMode = ref(localStorage.getItem('night-home-theme') !== 'dark')
const activeNav = ref('home')
function toggleLandingTheme() {
  lightMode.value = !lightMode.value
  localStorage.setItem('night-home-theme', lightMode.value ? 'light' : 'dark')
}
const entryPath = computed(() => props.isAuthenticated ? props.dashboardPath : '/login')
const features = [
  { key: 'gateway', icon: 'bolt' },
  { key: 'routing', icon: 'shield' },
  { key: 'usage', icon: 'chart' },
] as const
const steps = ['account', 'key', 'connect'] as const
const selectedModel = ref(1)
const demoModels = [{ name: 'Claude' }, { name: 'GPT' }, { name: 'Gemini' }]
const languages = [
  { id: 'curl', name: 'cURL' },
  { id: 'python', name: 'Python' },
  { id: 'node', name: 'Node.js' },
] as const
const selectedLanguage = ref<'curl' | 'python' | 'node'>('curl')
const shellQuote = (value: string) => "'" + value.replace(/'/g, "'\\''") + "'"
const sampleRequest = computed(() => {
  const base = (appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || window.location.origin)
    .replace(/\/+$/, '').replace(/\/v1$/, '')
  if (selectedModel.value === 0) {
    return {
      url: `${base}/v1/messages`,
      headers: { 'x-api-key': 'YOUR_API_KEY', 'anthropic-version': '2023-06-01', 'Content-Type': 'application/json' },
      body: { model: 'YOUR_CLAUDE_MODEL', max_tokens: 1024, messages: [{ role: 'user', content: 'Hello!' }] },
    }
  }
  if (selectedModel.value === 2) {
    return {
      url: `${base}/v1beta/models/YOUR_GEMINI_MODEL:generateContent`,
      headers: { 'x-goog-api-key': 'YOUR_API_KEY', 'Content-Type': 'application/json' },
      body: { contents: [{ parts: [{ text: 'Hello!' }] }] },
    }
  }
  return {
    url: `${base}/v1/responses`,
    headers: { Authorization: 'Bearer YOUR_API_KEY', 'Content-Type': 'application/json' },
    body: { model: 'YOUR_GPT_MODEL', input: 'Hello!' },
  }
})
const sampleCode = computed(() => {
  const { url, headers, body } = sampleRequest.value
  const jsonBody = JSON.stringify(body, null, 2)
  if (selectedLanguage.value === 'python') {
    return [
      'import json', 'from urllib.request import Request, urlopen', '',
      `payload = ${jsonBody}`, `headers = ${JSON.stringify(headers, null, 2)}`, '',
      'request = Request(', `    ${JSON.stringify(url)},`,
      '    data=json.dumps(payload).encode("utf-8"),', '    headers=headers,', '    method="POST",', ')', '',
      'with urlopen(request) as response:', '    print(json.load(response))',
    ].join('\n')
  }
  if (selectedLanguage.value === 'node') {
    return [
      `const response = await fetch(${JSON.stringify(url)}, {`, '  method: "POST",',
      `  headers: ${JSON.stringify(headers, null, 2).replace(/\n/g, '\n  ')},`,
      `  body: JSON.stringify(${jsonBody.replace(/\n/g, '\n  ')}),`, '});', '',
      'if (!response.ok) throw new Error(await response.text());', 'console.log(await response.json());',
    ].join('\n')
  }
  return [
    `curl ${shellQuote(url)} \\`,
    ...Object.entries(headers).map(([key, value]) => `  -H ${shellQuote(`${key}: ${value}`)} \\`),
    `  -d ${shellQuote(jsonBody)}`,
  ].join('\n')
})
</script>

<style scoped>
.night-home {
  --page: #101b32;
  --surface: #142039;
  --card: #1b2a45;
  --ink: #edf4ff;
  --muted: #9aacc8;
  --blue: #72a2ff;
  --line: rgba(128, 165, 232, .17);
  --soft-blue: rgba(96, 146, 247, .09);
  --shadow: 0 8px 24px rgba(3, 12, 31, .12);
  position: relative;
  isolation: isolate;
  overflow: hidden;
  min-height: 100vh;
  background: var(--page);
  color: var(--ink);
  font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}
.night-home.is-light {
  --page: #f0f7ff;
  --surface: #f9fcff;
  --card: rgba(255, 255, 255, .85);
  --ink: #101b33;
  --muted: #8191ad;
  --blue: #2463ff;
  --line: rgba(96, 150, 255, .13);
  --soft-blue: rgba(80, 137, 255, .045);
  --shadow: 0 5px 17px rgba(52, 114, 230, .065);
}
.shell { width: min(1160px, calc(100% - 96px)); margin-inline: auto; }
.ambient { position: absolute; z-index: -1; pointer-events: none; filter: blur(80px); border-radius: 50%; }
.ambient-one { width: 540px; height: 520px; right: -90px; top: 70px; background: rgba(135, 171, 255, .12); }
.ambient-two { width: 450px; height: 340px; left: -200px; top: 300px; background: rgba(143, 213, 255, .09); }
.home-header { min-height: 104px; display: flex; align-items: center; justify-content: space-between; gap: 26px; position: relative; z-index: 5; }
.brand { display: inline-flex; align-items: center; gap: 13px; flex-shrink: 0; }
.brand img, .brand-mark { width: 43px; height: 43px; flex-shrink: 0; border-radius: 12px; object-fit: contain; }
.brand-mark { display: grid; place-items: center; font-size: 35px; line-height: 1; color: #fff; background: radial-gradient(circle at 30% 70%, #66cfff, #3668d5 44%, #262a74 90%); box-shadow: inset 0 0 10px rgba(255,255,255,.3), 0 4px 10px #2463ff20; }
.brand > span:last-child { font-size: 23px; font-weight: 650; letter-spacing: .01em; }
.brand small { display: block; margin-top: 4px; color: var(--muted); font-size: 7px; font-weight: 500; letter-spacing: .13em; }
.desktop-nav { display: flex; gap: 34px; align-items: center; color: #637896; font-size: 13px; font-weight: 550; }
.desktop-nav a { position: relative; padding: 18px 0; transition: color .2s; white-space: nowrap; }
.desktop-nav a.active, .desktop-nav a:hover { color: var(--blue); }
.desktop-nav a.active::after { content: ""; position: absolute; bottom: 4px; height: 2px; border-radius: 2px; width: 26px; left: 50%; transform: translateX(-50%); background: #2463ff; }
.header-actions { display: flex; gap: 17px; align-items: center; }
.header-actions :deep(button) { color: #536a8f; }
.header-actions :deep(button:hover) { color: var(--blue); background: var(--soft-blue); }
.header-actions :deep(.absolute) { background: var(--surface); color: var(--ink); border-color: var(--line); }
.header-actions :deep(.absolute button) { color: var(--ink); }
.theme-button { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 6px; }
.header-login { min-width: 84px; padding: 11px 21px; border: 1px solid #6b94ff; border-radius: 11px; color: var(--blue); font-size: 13px; font-weight: 600; text-align: center; white-space: nowrap; transition: background .2s; }
.header-login:hover { background: var(--soft-blue); }
.hero { display: grid; grid-template-columns: 1fr 1fr; gap: 0; align-items: center; min-height: 620px; padding: 45px 0 77px; }
.hero-copy { position: relative; z-index: 2; padding: 25px 0; }
.eyebrow { display: inline-flex; align-items: center; gap: 10px; border: 1px solid var(--line); padding: 8px 15px; border-radius: 50px; box-shadow: 0 2px 5px rgba(62, 132, 255, .04); font-size: 11px; letter-spacing: .07em; color: #6585c1; background: rgba(255,255,255,.05); margin-bottom: 25px; }
.eyebrow > span { font-size: 19px; color: var(--blue); line-height: 1; }
.hero h1 { font-size: clamp(38px, 4.35vw, 58px); line-height: 1.38; font-weight: 680; letter-spacing: -.045em; }
.hero h1 em { color: #2463ff; font-style: normal; }
.hero-description { font-size: 14px; line-height: 2; color: #6f809c; white-space: pre-line; margin-top: 25px; max-width: 440px; }
.hero-actions { margin-top: 33px; display: flex; gap: 18px; }
.primary-button, .secondary-button { display: inline-flex; align-items: center; justify-content: center; gap: 25px; min-height: 50px; padding: 0 26px; border-radius: 9px; font-size: 12px; font-weight: 600; transition: transform .2s, box-shadow .2s; }
.primary-button { color: #fff; background: #2463ff; box-shadow: 0 5px 14px rgba(36,99,255,.2); }
.primary-button:hover { transform: translateY(-2px); box-shadow: 0 8px 20px rgba(36,99,255,.27); background: #1754ef; }
.secondary-button { color: var(--ink); border: 1px solid #b9d2ff; background: rgba(255,255,255,.1); gap: 10px; }
.secondary-button svg { color: var(--blue); }
.secondary-button:hover { background: var(--soft-blue); }
.hero-notes { margin-top: 30px; display: flex; flex-wrap: wrap; gap: 17px; color: var(--muted); font-size: 10px; }
.hero-notes > span { display: inline-flex; align-items: center; gap: 5px; }
.hero-notes svg { color: #5895ff; width: 12px; }
.hero-visual { position: relative; width: 100%; aspect-ratio: 1.07; min-width: 0; }
.orbit-art { pointer-events: none; position: absolute; inset: -12% -12%; width: 124%; max-width: none; height: 124%; object-fit: contain; }
.orbit-core { position: absolute; left: 50%; top: 51%; transform: translate(-50%, -50%); width: 143px; height: 143px; border: 1px solid rgba(255,255,255,.85); background: linear-gradient(135deg, rgba(255,255,255,.94),rgba(245,248,255,.58)); border-radius: 37px; display: flex; flex-direction: column; align-items: center; justify-content: center; color: #1647b4; box-shadow: 0 10px 38px #6aa0ff1a, inset 0 0 20px #fff; backdrop-filter: blur(12px); }
.core-spark { font: 49px/1 Georgia, serif; color: #2d73ff; text-shadow: 1px 3px 2px #7dc9ff, -1px -1px #fff, 0 4px 11px #6ca6ff9c; margin-bottom: 9px; }
.orbit-core strong { font-size: 21px; font-weight: 650; }
.orbit-core > span:last-child { margin-top: 9px; font: 7px monospace; color: #6490e0; letter-spacing: .04em; }
.provider-node { position: absolute; display: flex; gap: 13px; align-items: center; background: linear-gradient(125deg, rgba(255,255,255,.94),rgba(248,251,255,.7)); border: 1px solid rgba(255,255,255,.85); backdrop-filter: blur(12px); box-shadow: 0 12px 28px rgba(80,124,213,.1); border-radius: 23px; padding: 19px 20px; color: #1b2b4b; min-width: 164px; }
.provider-node > svg { flex-shrink: 0; }
.provider-node strong { font-size: 14px; font-weight: 600; display: block; }
.provider-node small { font-size: 10px; color: #8b9ab3; display: block; margin-top: 6px; }
.node-dot { width: 4px; height: 4px; margin-left: auto; flex-shrink: 0; border-radius: 50%; background: #b8d2ff; }
.node-claude { left: 0; top: 17%; }
.node-gpt { right: -4%; top: 29%; min-width: 146px; }
.node-gemini { left: 10%; bottom: 7%; }
.node-antigravity { right: -2%; bottom: 10%; min-width: 186px; }
.antigravity-mark { display: inline-block; font: 49px/.9 Georgia, serif; color: #26359d; text-shadow: 1px 0 #9ba2f7; }
.provider-node .antigravity-mark { font-size: 37px; }
.orbit-label { position: absolute; top: 65%; left: -2%; border: 1px solid rgba(255,255,255,.8); border-radius: 30px; background: rgba(255,255,255,.87); color: #5e7397; font-size: 8px; padding: 9px 11px; box-shadow: 0 4px 12px #729fff12; display: inline-flex; align-items: center; gap: 7px; }
.orbit-label > span { color: #2463ff; font-size: 12px; }
.content-surface { background: linear-gradient(180deg, var(--surface), var(--page) 50%, var(--surface)); border-radius: 16px 16px 0 0; padding-top: 66px; }
.models-section, .features-section, .start-section, #about { scroll-margin-top: 30px; }
.section-heading { display: flex; justify-content: space-between; gap: 25px; align-items: flex-end; }
.section-kicker { color: #7198e8; font-size: 11px; letter-spacing: .03em; margin-bottom: 14px; }
.section-kicker:not(.guide-kicker)::before { content: ""; height: 12px; width: 2px; display: inline-block; background: #a1bcff; margin-right: 8px; vertical-align: -1px; }
.section-heading h2, .guide-card h2 { font-size: 32px; line-height: 1.45; letter-spacing: -.035em; font-weight: 650; }
.text-link { color: var(--blue); font-size: 12px; font-weight: 600; display: inline-flex; align-items: center; gap: 14px; white-space: nowrap; }
.text-link:hover { color: #1745b8; }
.section-heading > .text-link { margin-bottom: 9px; }
.section-description { color: var(--muted); font-size: 13px; line-height: 1.9; margin-top: 19px; white-space: pre-line; }
.model-grid { display: grid; grid-template-columns: repeat(4, minmax(0,1fr)); gap: 24px; margin-top: 32px; }
.model-card { display: flex; align-items: center; gap: 19px; padding: 26px 24px; border: 1px solid var(--line); border-radius: 14px; background: var(--card); box-shadow: var(--shadow); transition: transform .2s, border-color .2s; }
.model-card:hover { transform: translateY(-3px); border-color: #7ea8ff; }
.model-card > svg { flex-shrink: 0; }
.model-card h3 { font-size: 14px; font-weight: 600; }
.model-card p { color: var(--muted); font-size: 10px; margin-top: 9px; }
.openai-badge { width: 40px; height: 40px; display: grid; place-items: center; flex-shrink: 0; border-radius: 11px; background: #20a187; }
.openai-badge :deep(path) { fill: white; }
.features-section { padding: 105px 0 75px; }
.heading-aside { max-width: 227px; white-space: pre-line; font-size: 12px; line-height: 1.9; color: var(--muted); }
.feature-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 23px; margin-top: 34px; }
.feature-card { padding: 28px 27px 22px; border: 1px solid var(--line); border-radius: 15px; background: var(--card); box-shadow: var(--shadow); transition: transform .2s, border-color .2s; display: flex; flex-direction: column; }
.feature-card:hover { transform: translateY(-4px); border-color: #8bb0ff; }
.feature-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.feature-icon { width: 57px; height: 57px; display: grid; place-items: center; border-radius: 14px; border: 1px solid rgba(99,154,255,.28); background: linear-gradient(145deg, var(--soft-blue),transparent); box-shadow: inset 0 0 0 1px rgba(99,154,255,.05); color: var(--blue); }
.feature-icon svg { width: 31px; height: 31px; }
.feature-number { font: 10px monospace; color: #9cb1d5; }
.feature-card h3 { font-size: 20px; font-weight: 620; line-height: 1.6; letter-spacing: -.02em; margin-bottom: 17px; }
.feature-card > p { font-size: 12px; line-height: 1.95; color: var(--muted); flex: 1; min-height: 76px; }
.feature-footer { border-top: 1px solid var(--line); padding-top: 21px; margin-top: 31px; font: 9px/1.6 monospace; color: #6e9eff; display: flex; align-items: center; gap: 8px; }
.feature-footer > span { width: 16px; height: 2px; background: #86aeff; flex-shrink: 0; }
.start-section { padding-bottom: 60px; }
.guide-card { position: relative; isolation: isolate; overflow: hidden; padding: 39px 36px 33px; background: linear-gradient(115deg, var(--soft-blue), transparent 55%, var(--soft-blue)); border: 1px solid var(--line); border-radius: 13px; box-shadow: var(--shadow); }
.guide-kicker { font: 11px monospace; letter-spacing: .04em; color: #7594cf; }
.guide-card h2 { position: relative; max-width: 80%; z-index: 1; }
.guide-card > .section-description { margin-top: 17px; }
.guide-card > .text-link { margin-top: 34px; }
.guide-art { position: absolute; pointer-events: none; right: 38px; top: 17px; width: 200px; height: 160px; z-index: -1; transform: rotate(-4deg); opacity: .9; }
.glass-window { position: absolute; width: 118px; height: 96px; border-radius: 12px; transform: skewY(18deg); box-shadow: 10px 12px 25px #5b9afa10; border: 1px solid #cbdefa8f; }
.window-back { top: 7px; right: 9px; background: linear-gradient(120deg,#d1edff88,#a8b8ff85); }
.window-middle { top: 20px; right: 31px; background: linear-gradient(145deg,#93cfff88,#8195ff97); }
.window-front { top: 35px; right: 49px; background: linear-gradient(145deg,#c0efffb3,#79a4ffb3); border-color: #b6d8ff; display: grid; place-items: center; }
.window-front::before { content: ""; position: absolute; left: 11px; top: 12px; width: 91px; height: 3px; background: #7ba6f77d; }
.window-front span { color: #fff; font: 36px monospace; text-shadow: 0 0 15px #fff; }
.art-cube { position: absolute; width: 18px; height: 18px; border-radius: 3px; transform: skewY(20deg) rotate(-12deg); background: linear-gradient(140deg,#e5faff,#a8bfff); box-shadow: 4px 5px 0 #d6e7ff; }
.cube-one { right: 4px; top: 112px; }
.cube-two { right: -27px; top: 37px; width: 25px; height: 25px; opacity: .4; }
.art-dot { position: absolute; left: 5px; bottom: 36px; width: 6px; height: 6px; border-radius: 50%; background: #b4d4ff; }
.steps { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 35px; margin-top: 43px; }
.steps li { display: flex; gap: 14px; }
.step-number { display: grid; place-items: center; width: 34px; height: 34px; flex-shrink: 0; border-radius: 50%; background: linear-gradient(135deg,#c9e4ff,#97b7ff); color: #1551d1; font: 11px monospace; }
.steps h3 { font-size: 13px; font-weight: 600; line-height: 1.5; margin-top: 5px; }
.steps p { font-size: 11px; color: var(--muted); line-height: 1.9; margin-top: 8px; }
.code-section { margin-top: 23px; padding: 27px 31px 23px; border-radius: 14px; background: var(--soft-blue); }
.code-section-heading { display: flex; justify-content: space-between; align-items: flex-start; gap: 18px; }
.code-section-heading h3 { font-size: 19px; font-weight: 600; }
.code-section-heading p { color: var(--muted); font-size: 11px; margin-top: 7px; }
.language-tabs { display: flex; align-items: center; gap: 5px; }
.language-tabs button { padding: 11px 21px; color: var(--muted); font-size: 11px; border-radius: 10px; transition: color .2s, background .2s; }
.language-tabs button.active { background: var(--card); color: var(--blue); box-shadow: 0 3px 12px #387bff09; font-weight: 600; }
.code-settings { display: flex; align-items: center; gap: 10px; font-size: 10px; color: var(--muted); margin: 20px 0 12px; }
.code-settings select { min-width: 86px; padding: 5px 25px 5px 9px; color: var(--ink); background-color: var(--card); border: 1px solid var(--line); border-radius: 5px; font: inherit; }
.code-settings > span { margin-left: auto; font-size: 9px; }
.code-panel { position: relative; min-width: 0; overflow: auto; border: 1px solid #364966; border-radius: 9px; background: linear-gradient(120deg,#182338,#25344c); color: #d5dfef; padding: 33px 24px 28px; min-height: 232px; box-shadow: inset 0 1px 0 #ffffff0a; }
.copy-button { position: sticky; left: calc(100% - 88px); top: 0; display: flex; align-items: center; gap: 5px; z-index: 2; border-radius: 5px; color: #c0cde1; font-size: 9px; margin-top: -20px; margin-bottom: 12px; padding: 6px 8px; background: #ffffff0c; }
.copy-button:hover { background: #ffffff17; }
.code-panel pre { font: 11px/2 "SFMono-Regular", Consolas, monospace; white-space: pre; margin: 0; }
.code-line { display: flex; }
.line-number { width: 30px; flex-shrink: 0; color: #8394ad; user-select: none; }
.code-footnote { display: flex; align-items: center; gap: 7px; font-size: 10px; line-height: 1.6; color: var(--muted); margin-top: 22px; }
.closing { position: relative; isolation: isolate; overflow: hidden; display: flex; align-items: center; gap: 30px; min-height: 155px; padding: 35px 43px; color: white; background: radial-gradient(ellipse at 70% 110%,#214ca8,#10265d 48%,#0c173c); border: 1px solid #4574d54d; border-radius: 19px; margin-bottom: 64px; box-shadow: 0 12px 30px #3464d50a; }
.closing::before { content: ""; pointer-events: none; position: absolute; inset: -100% -20%; z-index: -1; transform: rotate(-24deg); background: linear-gradient(180deg, transparent 43%,#2e6bff0a 46%,#81cfff5c 46.2%,#2463ff39 46.5%,transparent 48%,transparent 54%,#3167e822 55%,#76c5ff69 55.2%,#2463ff22 55.5%,transparent 57%); filter: blur(.5px); }
.closing::after { content: ""; position: absolute; z-index: -1; width: 90px; height: 6px; border-radius: 100%; background: #78c5ff; box-shadow: 0 0 15px 4px #6b9dff; left: 66%; bottom: 27%; transform: rotate(-24deg); filter: blur(4px); opacity: .65; }
.closing-star { font: 53px/1 Georgia,serif; text-shadow: 0 0 15px #c3f2ffb3; }
.closing h2 { font-size: 25px; font-weight: 600; letter-spacing: -.025em; }
.closing p { color: #bacced; font-size: 12px; line-height: 1.8; margin-top: 10px; }
.closing .primary-button { margin-left: auto; white-space: nowrap; background: #fff; color: #2463ff; min-height: 52px; border: 1px solid #b6cbff; box-shadow: 0 0 18px #7ca9ff26; }
.home-footer { color: #b5bfd4; background: linear-gradient(110deg,#111c30,#19273d); }
.footer-top { display: flex; justify-content: space-between; align-items: center; gap: 24px; padding: 30px 0 27px; border-bottom: 1px solid #839ac02b; }
.home-footer .brand { gap: 11px; color: #e8efff; }
.home-footer .brand img,.home-footer .brand-mark { width: 38px; height: 38px; border-radius: 10px; }
.home-footer .brand > span:last-child { font-size: 17px; }
.home-footer .brand small { font-size: 6px; color: #8c9cba; }
.footer-top nav { display: flex; gap: 29px; font-size: 11px; }
.footer-top a:hover, .footer-bottom a:hover { color: #fff; }
.footer-bottom { display: flex; justify-content: space-between; align-items: center; gap: 20px; min-height: 78px; color: #90a2bf; font-size: 10px; }
.night-home:not(.is-light) .hero-description { color: var(--muted); }
.night-home:not(.is-light) .orbit-art { opacity: .8; }
.night-home:not(.is-light) .hero h1 em { color: #81acff; }
.night-home:not(.is-light) .model-card .antigravity-mark { color: #adbcff; }
.night-home a:focus-visible, .night-home button:focus-visible, .night-home select:focus-visible { outline: 2px solid var(--blue); outline-offset: 5px; border-radius: 5px; }
@media (min-width: 1500px) { .hero { min-height: 670px; } }
@media (max-width: 1100px) {
  .shell { width: calc(100% - 72px); }
  .desktop-nav { gap: 24px; font-size: 12px; }
  .header-actions { gap: 11px; }
  .brand > span:last-child { font-size: 21px; }
  .brand img,.brand-mark { width: 39px; height: 39px; }
  .brand small { font-size: 6px; }
  .header-login { min-width: 73px; padding: 10px 17px; }
  .hero { min-height: 570px; padding: 38px 0 65px; }
  .hero h1 { font-size: 45px; }
  .hero-description { font-size: 13px; }
  .hero-visual { aspect-ratio: 1; }
  .orbit-core { width: 119px; height: 119px; border-radius: 30px; }
  .orbit-core strong { font-size: 18px; }
  .core-spark { font-size: 39px; }
  .provider-node { padding: 15px; min-width: 145px; border-radius: 20px; gap: 10px; }
  .provider-node strong { font-size: 12px; }
  .provider-node small { font-size: 9px; }
  .node-gpt { min-width: 129px; }
  .node-antigravity { min-width: 157px; }
  .model-grid { gap: 18px; }
  .model-card { padding: 24px 19px; gap: 14px; }
  .model-card h3 { font-size: 13px; }
  .model-card > svg { width: 37px; height: 37px; }
  .feature-card { padding: 25px 24px; }
  .feature-card h3 { font-size: 19px; }
  .feature-footer { font-size: 8px; gap: 6px; }
  .section-heading h2,.guide-card h2 { font-size: 29px; }
}
@media (max-width: 900px) {
  .shell { width: calc(100% - 64px); }
  .home-header { min-height: 90px; gap: 15px; }
  .brand { gap: 10px; }
  .brand > span:last-child { font-size: 19px; }
  .brand small { font-size: 5px; letter-spacing: .1em; }
  .brand img,.brand-mark { width: 36px; height: 36px; }
  .desktop-nav { gap: 20px; font-size: 10px; }
  .header-actions { gap: 7px; }
  .header-actions :deep(button) { font-size: 11px; }
  .header-login { min-width: 68px; font-size: 11px; padding: 9px 15px; border-radius: 9px; }
  .theme-button { width: 25px; }
  .hero { min-height: 490px; padding: 28px 0 61px; }
  .eyebrow { font-size: 10px; padding: 6px 12px; gap: 8px; margin-bottom: 21px; }
  .eyebrow > span { font-size: 17px; }
  .hero h1 { font-size: 37px; line-height: 1.4; }
  .hero-description { font-size: 12px; line-height: 2; margin-top: 20px; }
  .hero-actions { margin-top: 28px; gap: 12px; }
  .primary-button,.secondary-button { min-height: 45px; font-size: 11px; padding: 0 20px; gap: 18px; }
  .secondary-button { gap: 9px; }
  .hero-notes { font-size: 9px; gap: 12px; margin-top: 27px; }
  .hero-notes svg { width: 10px; }
  .orbit-core { width: 105px; height: 105px; border-radius: 26px; }
  .orbit-core strong { font-size: 16px; }
  .orbit-core > span:last-child { font-size: 6px; margin-top: 7px; }
  .core-spark { font-size: 34px; margin-bottom: 7px; }
  .provider-node { min-width: 125px; padding: 13px; gap: 10px; border-radius: 18px; }
  .provider-node > svg { width: 28px; height: 28px; }
  .provider-node strong { font-size: 11px; }
  .provider-node small { font-size: 8px; }
  .node-gpt { min-width: 115px; right: -3%; }
  .node-claude { left: -2%; }
  .node-gemini { left: 7%; }
  .node-antigravity { min-width: 135px; right: -1%; }
  .provider-node .antigravity-mark { font-size: 30px; }
  .node-dot { width: 3px; height: 3px; }
  .orbit-label { font-size: 7px; padding: 7px 9px; left: -3%; }
  .content-surface { padding-top: 48px; }
  .section-heading h2,.guide-card h2 { font-size: 25px; }
  .section-kicker { font-size: 10px; margin-bottom: 12px; }
  .section-description { font-size: 12px; margin-top: 17px; }
  .section-heading > .text-link { margin-bottom: 6px; }
  .text-link { font-size: 11px; gap: 12px; }
  .model-grid { gap: 15px; margin-top: 27px; }
  .model-card { gap: 13px; padding: 22px 17px; border-radius: 12px; }
  .model-card > svg { width: 34px; height: 34px; }
  .model-card h3 { font-size: 12px; }
  .model-card p { font-size: 9px; margin-top: 7px; }
  .openai-badge { width: 33px; height: 33px; border-radius: 9px; }
  .openai-badge :deep(svg) { width: 25px; height: 25px; }
  .antigravity-mark { font-size: 37px; }
  .features-section { padding: 78px 0 58px; }
  .heading-aside { font-size: 11px; max-width: 190px; }
  .feature-grid { gap: 15px; margin-top: 29px; }
  .feature-card { padding: 23px 20px 20px; border-radius: 13px; }
  .feature-icon { width: 46px; height: 46px; border-radius: 12px; }
  .feature-icon svg { width: 27px; height: 27px; }
  .feature-top { margin-bottom: 26px; }
  .feature-card h3 { font-size: 17px; margin-bottom: 15px; }
  .feature-card > p { font-size: 11px; min-height: 83px; }
  .feature-footer { font-size: 7px; padding-top: 18px; margin-top: 25px; }
  .feature-footer > span { width: 12px; }
  .guide-card { padding: 31px 28px 29px; }
  .guide-card h2 { max-width: 87%; }
  .guide-art { width: 170px; height: 150px; right: 16px; top: 7px; opacity: .65; transform: scale(.88); }
  .steps { gap: 21px; margin-top: 37px; }
  .steps li { gap: 11px; }
  .step-number { width: 29px; height: 29px; font-size: 10px; }
  .steps h3 { font-size: 11px; margin-top: 4px; }
  .steps p { font-size: 10px; }
  .guide-card > .text-link { margin-top: 29px; }
  .code-section { padding: 24px 25px 22px; margin-top: 19px; }
  .code-section-heading h3 { font-size: 17px; }
  .code-section-heading p { font-size: 10px; }
  .language-tabs button { padding: 10px 18px; font-size: 10px; }
  .code-panel { min-height: 215px; padding: 28px 20px 23px; }
  .code-panel pre { font-size: 9px; }
  .code-settings { margin-top: 15px; font-size: 9px; }
  .code-settings > span { font-size: 8px; }
  .code-footnote { font-size: 9px; margin-top: 17px; }
  .start-section { padding-bottom: 43px; }
  .closing { min-height: 130px; padding: 29px 33px; margin-bottom: 42px; gap: 26px; border-radius: 15px; }
  .closing-star { font-size: 44px; }
  .closing h2 { font-size: 21px; }
  .closing p { font-size: 11px; margin-top: 8px; }
  .closing .primary-button { min-height: 46px; }
  .footer-top { padding: 25px 0 23px; }
  .home-footer .brand > span:last-child { font-size: 14px; }
  .home-footer .brand img,.home-footer .brand-mark { width: 32px; height: 32px; border-radius: 8px; }
  .home-footer .brand small { font-size: 5px; }
  .footer-top nav { gap: 24px; font-size: 10px; }
  .footer-bottom { min-height: 70px; font-size: 9px; }
  .footer-bottom > span:nth-child(2) { display: none; }
}
@media (max-width: 700px) {
  .shell { width: calc(100% - 40px); }
  .desktop-nav { display: none; }
  .home-header { min-height: 80px; }
  .brand > span:last-child { font-size: 19px; }
  .brand small { font-size: 6px; }
  .header-actions { gap: 9px; }
  .hero { grid-template-columns: 1fr; padding: 25px 0 42px; gap: 20px; }
  .hero-copy { text-align: center; padding: 12px 0; }
  .eyebrow { margin-bottom: 20px; }
  .hero h1 { font-size: clamp(35px, 8.2vw, 48px); }
  .hero-description { font-size: 12px; margin: 18px auto 0; max-width: 380px; }
  .hero-actions { justify-content: center; gap: 14px; margin-top: 24px; }
  .hero-notes { justify-content: center; margin-top: 22px; gap: 15px; }
  .hero-visual { width: 93%; max-width: 460px; margin: 0 auto; aspect-ratio: 1.05; }
  .provider-node { padding: 14px; min-width: 134px; }
  .provider-node strong { font-size: 12px; }
  .provider-node small { font-size: 9px; }
  .node-antigravity { min-width: 148px; }
  .node-gpt { min-width: 126px; right: -2%; }
  .node-claude { left: -1%; }
  .orbit-core { width: 112px; height: 112px; border-radius: 29px; }
  .orbit-core strong { font-size: 17px; }
  .orbit-label { left: -1%; font-size: 7px; }
  .content-surface { padding-top: 37px; }
  .section-heading { gap: 13px; align-items: flex-start; }
  .section-heading h2,.guide-card h2 { font-size: 23px; }
  .section-heading > .text-link { font-size: 10px; align-self: flex-end; margin-bottom: 5px; }
  .section-heading > .text-link svg { width: 14px; }
  .section-description { font-size: 11px; }
  .model-grid { grid-template-columns: repeat(2,minmax(0,1fr)); gap: 12px; margin-top: 23px; }
  .model-card { padding: 22px 21px; gap: 17px; }
  .model-card h3 { font-size: 13px; }
  .model-card p { font-size: 9px; }
  .models-section .section-heading { flex-wrap: wrap; }
  .features-section { padding: 55px 0 35px; }
  .features-section .section-heading { display: block; }
  .heading-aside { margin-top: 13px; max-width: none; white-space: normal; font-size: 11px; }
  .feature-grid { grid-template-columns: 1fr; gap: 14px; margin-top: 23px; }
  .feature-card { padding: 24px; }
  .feature-top { margin-bottom: 20px; }
  .feature-card h3 { font-size: 19px; margin-bottom: 13px; }
  .feature-card > p { min-height: 0; font-size: 12px; }
  .feature-footer { font-size: 8px; margin-top: 23px; padding-top: 17px; }
  .guide-card { padding: 27px 24px; }
  .guide-card h2 { max-width: 100%; font-size: 24px; line-height: 1.6; }
  .guide-card > .section-description { max-width: 78%; }
  .guide-art { right: -14px; top: 7px; opacity: .3; transform: scale(.65); transform-origin: top right; }
  .steps { grid-template-columns: 1fr; gap: 22px; margin-top: 30px; }
  .steps li { gap: 13px; }
  .steps h3 { font-size: 12px; }
  .steps p { font-size: 11px; margin-top: 6px; }
  .step-number { width: 31px; height: 31px; }
  .guide-card > .text-link { margin-top: 28px; }
  .code-section { padding: 22px 16px; }
  .code-section-heading { gap: 12px; flex-wrap: wrap; }
  .code-section-heading h3 { font-size: 17px; }
  .code-section-heading p { font-size: 10px; }
  .language-tabs { gap: 1px; }
  .language-tabs button { padding: 9px 14px; font-size: 10px; }
  .code-settings { flex-wrap: wrap; gap: 7px; }
  .code-settings > span { width: 100%; margin: 0 0 1px; font-size: 8px; }
  .code-panel { padding: 27px 12px 21px; min-height: 209px; }
  .code-panel pre { font-size: 9px; }
  .line-number { width: 24px; }
  .copy-button { font-size: 8px; padding: 5px 6px; }
  .code-footnote { font-size: 9px; align-items: flex-start; }
  .code-footnote svg { flex-shrink: 0; margin-top: 2px; }
  .closing { flex-wrap: wrap; gap: 15px; padding: 27px 23px; margin-bottom: 35px; }
  .closing-star { font-size: 36px; }
  .closing h2 { font-size: 19px; }
  .closing p { font-size: 10px; }
  .closing .primary-button { margin: 8px 0 0 48px; min-height: 43px; }
  .footer-top { flex-wrap: wrap; gap: 22px; padding: 25px 0 22px; }
  .footer-top nav { width: 100%; gap: 27px; font-size: 10px; }
  .footer-bottom { min-height: 66px; font-size: 8px; gap: 8px; }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { transition: none !important; }
}
</style>
