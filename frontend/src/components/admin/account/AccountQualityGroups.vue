<template>
  <section class="quality-workspace">
    <header class="quality-header">
      <div class="min-w-0">
        <h2 class="quality-title">{{ t('accountQuality.title') }}</h2>
        <p class="quality-subtitle">{{ t('accountQuality.subtitle') }}</p>
      </div>
      <div class="quality-header-actions">
        <slot name="view-switch" />
        <label class="quality-auto-refresh">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': autoRefreshFetching }" />
          <select
            :value="autoRefreshEnabled ? autoRefreshIntervalSeconds : 0"
            :aria-label="t('accountQuality.autoRefresh')"
            @change="changeAutoRefresh"
          >
            <option :value="0">{{ t('accountQuality.autoRefreshOff') }}</option>
            <option v-for="seconds in autoRefreshIntervals" :key="seconds" :value="seconds">
              {{ t('accountQuality.autoRefreshEvery', { seconds }) }}
            </option>
          </select>
          <span v-if="autoRefreshEnabled" class="tabular-nums">{{ autoRefreshCountdown }}s</span>
        </label>
        <button class="quality-button quality-button--secondary" :disabled="loading" @click="refresh">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          {{ t('accountQuality.refresh') }}
        </button>
        <button class="quality-button quality-button--primary" @click="emit('create')">
          <Icon name="plus" size="sm" />
          {{ t('accountQuality.create') }}
        </button>
      </div>
    </header>
    <div class="quality-overview">
      <div
        v-for="stat in overview"
        :key="stat.label"
        class="quality-stat"
      >
        <span class="quality-stat-icon" :class="stat.tone"><Icon :name="stat.icon" size="md" /></span>
        <div class="min-w-0">
          <p class="quality-stat-label">{{ stat.label }}</p>
          <p class="quality-stat-value">{{ stat.value }}</p>
        </div>
      </div>
    </div>
    <div class="quality-toolbar">
      <label class="quality-search">
        <Icon name="search" size="sm" />
        <input
          v-model="search"
          :aria-label="t('accountQuality.search')"
          :placeholder="t('accountQuality.search')"
        />
      </label>
      <input
        v-model.trim="metricsModel"
        type="search"
        class="quality-select quality-model-filter"
        :aria-label="t('accountQuality.metricsModel')"
        :placeholder="t('accountQuality.metricsModelPlaceholder')"
      />
      <select
        v-model="benchmark"
        class="quality-select quality-benchmark"
        :aria-label="t('accountQuality.benchmark')"
      >
        <option value="candy">{{ t('accountQuality.benchmarkCandy') }}</option>
        <option value="modeltrace">
          {{ t('accountQuality.benchmarkTrace') }}
        </option>
        <option value="context">
          {{ t('accountQuality.benchmarkContext') }}
        </option>
      </select>
      <select v-model="platform" class="quality-select" :aria-label="t('accountQuality.all')">
        <option value="">{{ t('accountQuality.all') }}</option>
        <option v-for="p in platforms" :key="p" :value="p">
          {{ p }}
        </option></select
      ><select
        v-model="status"
        class="quality-select"
        :aria-label="t('accountQuality.statusFilter')"
      >
        <option value="">{{ t('accountQuality.allStatuses') }}</option>
        <option value="available">{{ t('accountQuality.active') }}</option>
        <option value="unavailable">{{ t('accountQuality.disabled') }}</option>
        <option value="error">{{ t('accountQuality.errorStatus') }}</option>
      </select>
      <select v-model="sort" class="quality-select" :aria-label="t('accountQuality.sortScore')">
        <option value="score">{{ t('accountQuality.sortScore') }}</option>
        <option value="ttft">{{ t('accountQuality.sortTTFT') }}</option>
        <option value="cache">{{ t('accountQuality.sortCache') }}</option>
        <option value="name">{{ t('accountQuality.sortName') }}</option>
      </select>
    </div>
    <div class="quality-context">
      <span class="quality-context-label"><Icon name="infoCircle" size="sm" />{{ t('accountQuality.policyModes') }}</span>
      <HelpTooltip :content="t('accountQuality.observeHint')" />
      <template v-if="benchmark === 'candy'">
        <span class="quality-context-divider" aria-hidden="true"></span>
        <span>{{ t('accountQuality.accountCredentials') }}</span>
        <HelpTooltip :content="t('accountQuality.manxueScope')" />
      </template>
    </div>
    <p
      v-if="error"
      role="alert"
      class="mb-4 rounded-lg bg-red-50 p-3 text-sm text-red-700"
    >
      {{ error }}
    </p>
    <p
      v-if="notice"
      role="status"
      class="mb-4 rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700"
    >
      {{ notice }}
    </p>
    <div v-if="loading" class="py-10 text-center text-gray-500">
      {{ t('accountQuality.loading') }}
    </div>
    <div
      v-else-if="!visibleGroups.length"
      class="rounded-xl border border-dashed p-12 text-center text-gray-500"
    >
      {{ t('accountQuality.empty') }}
    </div>
    <div v-else class="quality-groups">
      <article
        v-for="g in visibleGroups"
        :key="g.id"
        class="quality-group"
      >
        <header class="quality-group-header">
          <input
            type="checkbox"
            :checked="isGroupPageSelected(g.id)"
            :disabled="!pages[g.id]?.items.length"
            :aria-label="
              t('accountQuality.selectGroupPage', { name: groupName(g) })
            "
            @change="toggleGroupSelection(g.id)"
          /><button
            class="quality-group-title"
            :aria-expanded="!collapsed[g.id]"
            @click="toggle(g)"
          >
            <span
              class="quality-group-icon"
              ><Icon name="grid" size="sm" /></span
            ><span
              ><span
                class="quality-group-name"
                >{{ groupName(g) }}</span
              ><span class="quality-group-meta"
                ><span class="quality-platform">{{ g.platform }}</span>
                <span>{{ g.total }} {{ t('accountQuality.account') }}</span>
                <span class="quality-available"><span class="quality-status-dot"></span>{{ g.available }} {{ t('accountQuality.available') }}</span></span
              ></span
            >
          </button>
          <div class="quality-group-actions">
            <span class="quality-schedule" :class="{ 'quality-schedule--active': g.policy.enabled }"><Icon name="clock" size="xs" />{{
              g.policy.enabled
                ? g.policy.interval_minutes +
                  ' ' +
                  t('accountQuality.minutes') +
                  ' · ' +
                  date(g.next_run_at)
                : t('accountQuality.notScheduled')
            }}</span
            ><template v-if="g.id"
              ><button
                class="quality-button quality-button--small quality-button--secondary"
                @click="openSettings(g)"
              >
                <Icon name="cog" size="xs" />
                {{ t('accountQuality.settings') }}</button
              ><button
                class="quality-button quality-button--small quality-button--soft"
                :disabled="!!activeRun"
                @click="requestRun(g)"
              >
                <Icon name="play" size="xs" />
                {{ t('accountQuality.inspect') }}
              </button></template
            ><button
              v-if="g.id"
              class="quality-button quality-button--small quality-button--ghost"
              @click="openRouting(g)"
            >
              <Icon name="chart" size="xs" />
              {{ t('accountQuality.routingDiagnostics') }}</button
            ><button
              v-if="g.id"
              class="quality-button quality-button--small quality-button--ghost"
              @click="openHistory(g, 0)"
            >
              <Icon name="clock" size="xs" />
              {{ t('accountQuality.history') }}
            </button>
            <span class="quality-action-divider" aria-hidden="true"></span>
            <button
              class="quality-button quality-button--small quality-button--ghost quality-button--icon"
              :title="t(collapsed[g.id] ? 'accountQuality.expand' : 'accountQuality.collapse')"
              :aria-expanded="!collapsed[g.id]"
              @click="toggle(g)"
            >
              <Icon name="chevronDown" size="sm" class="transition-transform" :class="{ '-rotate-90': collapsed[g.id] }" />
              <span class="sr-only">{{ t(collapsed[g.id] ? 'accountQuality.expand' : 'accountQuality.collapse') }}</span>
            </button>
          </div>
        </header>
        <div v-if="!collapsed[g.id]">
          <div
            class="quality-group-context"
          >
            <span
              >{{ t('accountQuality.window') }} ·
              {{ metricsModel || t('accountQuality.allModels') }}
              <span v-if="g.policy.model" class="ml-3">
                · {{ t('accountQuality.inspectionScope') }}: {{ g.policy.model }} / {{ g.policy.effort }}
              </span></span
            ><span
              >{{ t('accountQuality.mode.' + g.policy.mode)
              }}<span v-if="pages[g.id]?.summary" class="ml-3">
                ·
                {{
                  t('accountQuality.groupSummary', {
                    scored: pages[g.id].summary.scored,
                    pending: pages[g.id].summary.pending,
                    review: pages[g.id].summary.review
                  })
                }}</span
              ></span
            >
          </div>
          <p
            v-if="rowErrors[g.id]"
            role="alert"
            class="p-5 text-sm text-red-600"
          >
            {{ rowErrors[g.id] }}
            <button class="underline" @click="load(g)">
              {{ t('accountQuality.refresh') }}
            </button>
          </p>
          <div
            v-else-if="busy[g.id] && !pages[g.id]"
            class="p-8 text-center text-sm text-gray-500"
          >
            {{ t('accountQuality.loading') }}
          </div>
          <div v-else class="quality-table-scroll">
            <table class="quality-table">
              <thead class="text-[11px] uppercase tracking-wide text-gray-400">
                <tr>
                  <th class="px-5 py-3">{{ t('accountQuality.account') }}</th>
                  <th class="px-3 py-3">{{ t('admin.accounts.columns.capacity') }}</th>
                  <th class="px-3 py-3" :title="t('accountQuality.schedulingHint')">{{ t('admin.accounts.columns.schedulable') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.candy') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.ttft') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.cache') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.stability') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.score') }}</th>
                  <th class="px-3 py-3">{{ t('accountQuality.weight') }}</th>
                  <th class="px-5 py-3 text-right">
                    {{ t('accountQuality.actions') }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr
                  v-for="a in pages[g.id]?.items || []"
                  :key="a.id"
                  :class="{ 'quality-row--selected': props.selectedIds.includes(a.id) }"
                >
                  <td class="px-5 py-4">
                    <div
                      class="quality-account-identity flex items-center gap-2"
                      :class="{
                        'quality-account-identity--error': a.status === 'error',
                        'quality-account-identity--disabled': a.status !== 'error' && isAccountUnavailable(a)
                      }"
                    >
                      <input
                        type="checkbox"
                        :checked="props.selectedIds.includes(a.id)"
                        :aria-label="
                          t('accountQuality.selectAccount', { name: a.name })
                        "
                        @change="toggleSelection(a)"
                      />
                      <span
                        :class="
                          a.status === 'error'
                            ? 'bg-red-500'
                            : isAccountUnavailable(a)
                              ? 'bg-yellow-500'
                              : 'bg-emerald-500'
                        "
                        class="h-1.5 w-1.5 rounded-full"
                      ></span>
                      <HelpTooltip
                        v-if="accountHomepageUrl(a)"
                        :content="accountHomepageUrl(a)"
                        width-class="w-max max-w-sm break-all"
                        class="min-w-0"
                      >
                        <template #trigger>
                          <a
                            :href="accountHomepageUrl(a)"
                            target="_blank"
                            rel="noopener noreferrer"
                            class="quality-account-link"
                          >{{ a.name }}</a>
                        </template>
                      </HelpTooltip>
                      <span
                        v-else
                        class="quality-account-name"
                        :title="a.name"
                        >{{ a.name }}</span
                      >
                    </div>
                    <div class="quality-account-meta">
                      #{{ a.id }} · {{ a.type }}
                      <AccountStatusIndicator
                        class="mt-1 whitespace-normal"
                        :account="{ ...a.capacity, ...a }"
                        @show-temp-unsched="emit('show-temp-unsched', a.id)"
                      />
                    </div>
                  </td>
                  <td class="px-3 py-4" :title="t('accountQuality.capacityHint')">
                    <AccountCapacityCell v-if="a.capacity" :account="a.capacity" />
                    <span v-else class="font-mono text-xs text-gray-400">— / {{ a.concurrency }}</span>
                  </td>
                  <td class="px-3 py-4">
                    <button
                      role="switch"
                      :aria-checked="a.schedulable"
                      :aria-label="t('accountQuality.toggleScheduling', { name: a.name })"
                      :aria-busy="!!scheduling[a.id]"
                      :disabled="!!scheduling[a.id]"
                      :title="t('accountQuality.schedulingHint')"
                      class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-wait disabled:opacity-50 dark:focus:ring-offset-dark-800"
                      :class="a.schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'"
                      @click="toggleScheduling(a)"
                    >
                      <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform" :class="a.schedulable ? 'translate-x-4' : 'translate-x-0'" />
                    </button>
                    <p class="mt-1 text-[10px] text-gray-500">{{ t(a.schedulable ? 'admin.accounts.schedulableEnabled' : 'admin.accounts.schedulableDisabled') }}</p>
                    <p v-if="schedulingErrors[a.id]" role="alert" class="mt-1 max-w-40 whitespace-normal text-xs text-red-600">{{ schedulingErrors[a.id] }}</p>
                  </td>
                  <td class="px-3 py-4">
                    <button
                      class="quality-verdict"
                      :class="verdictClass(a.candy_status)"
                      @click="openHistory(g, a.id, a.evidence_ids)"
                    >
                      {{ verdict(a.candy_status) }}
                    </button>
                    <div
                      v-if="a.trace_status && a.trace_status !== 'untested'"
                      class="mt-1 text-[10px] text-gray-500"
                    >
                      {{ t('accountQuality.trace') }}:
                      {{ verdict(a.trace_status)
                      }}<span v-if="a.trace_candidate">
                        · {{ a.trace_candidate }}</span
                      >
                    </div>
                    <div
                      v-if="a.last_test_at"
                      class="mt-1 text-[10px] text-gray-400"
                    >
                      <span v-if="a.quality_samples">{{ a.quality_passed }}/{{ a.quality_samples }} · </span>
                      {{ date(a.last_test_at) }}
                    </div>
                  </td>
                  <td class="px-3 py-4">
                    <div class="font-mono text-xs">
                      {{ latency(a.p50_ms) }} / {{ latency(a.p95_ms) }}
                    </div>
                    <div
                      class="mt-1 text-[10px] text-gray-400"
                      :title="t('accountQuality.sampleHint')"
                    >
                      {{ t('accountQuality.requestsCount') }} {{ a.requests }} ·
                      {{ t('accountQuality.validTTFTCount') }} {{ a.ttft_samples }}
                    </div>
                  </td>
                  <td class="px-3 py-4">
                    <div class="font-mono text-xs">{{ percent(a.cache_rate) }}</div>
                    <div
                      class="mt-1 text-[10px] text-gray-400"
                      :title="t('accountQuality.cacheSampleHint')"
                    >
                      {{ t('accountQuality.requestsCount') }} {{ a.cache_samples ?? 0 }}
                    </div>
                  </td>
                  <td class="px-3 py-4">
                    <div class="font-mono text-xs">
                      {{ percent(a.success_rate) }}
                    </div>
                    <div class="mt-1 text-[10px] text-gray-400">
                      {{ t('accountQuality.successCount') }} {{ a.requests }} ·
                      {{ t('accountQuality.failureCount') }} {{ a.failures
                      }}<span v-if="!a.reliability_known">
                        · {{ t('accountQuality.coverageUnknown') }}</span
                      >
                    </div>
                  </td>
                  <td class="px-3 py-4">
                    <button
                      v-if="a.score !== null"
                      class="quality-score"
                      :title="t('accountQuality.explanation')"
                      @click="explanation = a"
                      >{{ a.score.toFixed(1) }}</button
                    ><span v-else class="text-xs text-gray-400">{{
                      t('accountQuality.noData')
                    }}</span>
                  </td>
                  <td class="px-3 py-4 font-mono text-xs text-gray-500">
                    {{ percent(a.suggested_weight) }}
                    <div
                      v-if="g.policy.mode === 'auto'"
                      class="mt-1 text-[10px] text-primary-600"
                    >
                      {{
                        a.effective_multiplier == null
                          ? t('accountQuality.notApplied')
                          : a.effective_multiplier.toFixed(2) + '×'
                      }}
                    </div>
                  </td>
                  <td class="px-5 py-4">
                    <div class="quality-row-actions">
                      <button
                        class="quality-button quality-button--small quality-button--ghost quality-button--icon"
                        :title="t('accountQuality.edit')"
                        @click="emit('edit', a.id)"
                      >
                        <Icon name="edit" size="sm" />
                        <span class="sr-only">{{ t('accountQuality.edit') }}</span></button
                      ><button
                        class="quality-button quality-button--small quality-button--ghost quality-button--icon"
                        :title="t('accountQuality.test')"
                        @click="emit('test', a.id)"
                      >
                        <Icon name="link" size="sm" />
                        <span class="sr-only">{{ t('accountQuality.test') }}</span></button
                      ><button
                        v-if="g.id"
                        :disabled="
                          !!activeRun || a.status !== 'active' || !a.schedulable ||
                          !canInspect(a, g.policy)
                        "
                        :title="inspectionHint(a, g.policy)"
                        class="quality-button quality-button--small quality-button--soft"
                        @click="requestRun(g, a.id)"
                      >
                        <Icon name="play" size="xs" />
                        {{
                          t(
                            benchmark === 'modeltrace'
                              ? 'accountQuality.benchmarkTrace'
                              : benchmark === 'context'
                                ? 'accountQuality.benchmarkContext'
                                : 'accountQuality.candy'
                          )
                        }}</button
                      ><button
                        class="quality-button quality-button--small quality-button--ghost quality-button--icon"
                        :title="t('accountQuality.history')"
                        @click="openHistory(g, a.id, a.evidence_ids)"
                      >
                        <Icon name="clock" size="sm" />
                        <span class="sr-only">{{ t('accountQuality.history') }}</span>
                      </button>
                    </div>
                    <p v-if="benchmark === 'candy' && a.type === 'oauth'" class="quality-row-hint">
                      <HelpTooltip :content="t('accountQuality.oauthUnsupported')" />
                      {{ t('accountQuality.oauthUnsupportedShort') }}
                    </p>
                  </td>
                </tr>
              </tbody>
            </table>
            <p
              v-if="!pages[g.id]?.items.length && !busy[g.id]"
              class="p-6 text-center text-sm text-gray-400"
            >
              {{ t('accountQuality.noRows') }}
            </p>
          </div>
          <footer class="quality-group-footer">
            <span
              >{{ pages[g.id]?.total || 0 }} ·
              {{ t('accountQuality.account') }}</span
            >
            <div class="quality-pagination">
              <button
                :disabled="busy[g.id] || (pages[g.id]?.page || 1) <= 1"
                class="quality-button quality-button--small quality-button--secondary"
                @click="load(g, (pages[g.id]?.page || 1) - 1)"
              >
                <Icon name="chevronLeft" size="xs" />
                {{ t('accountQuality.previous') }}</button
              ><span
                >{{ pages[g.id]?.page || 1 }} /
                {{
                  Math.max(1, Math.ceil((pages[g.id]?.total || 0) / 8))
                }}</span
              ><button
                :disabled="
                  busy[g.id] ||
                  (pages[g.id]?.page || 1) * 8 >= (pages[g.id]?.total || 0)
                "
                class="quality-button quality-button--small quality-button--secondary"
                @click="load(g, (pages[g.id]?.page || 1) + 1)"
              >
                {{ t('accountQuality.next') }}
                <Icon name="chevronRight" size="xs" />
              </button>
            </div>
          </footer>
        </div>
      </article>
    </div>
    <p class="mt-4 text-xs leading-5 text-gray-400">
      {{ t('accountQuality.metricHint') }}
    </p>
    <div
      v-if="activeRun"
      class="sticky bottom-4 mt-4 flex items-center justify-between rounded-xl border border-primary-200 bg-primary-50 px-5 py-3 text-sm text-primary-800 shadow-lg dark:bg-dark-700 dark:text-white"
    >
      <span
        >{{ verdict(activeRun.status) }} ·
        {{ t('accountQuality.runProgress') }} {{ activeRun.completed }} / ≤{{
          activeRun.planned
        }}</span
      ><button class="underline" @click="cancelRound">
        {{ t('accountQuality.cancelRun') }}
      </button>
    </div>
    <Teleport to="body"
      ><div
        v-if="settingsGroup || historyGroup || pendingRun"
        class="fixed inset-0 z-[70] flex items-center justify-center bg-black/40 p-4"
        @click.self="closeModal"
      >
        <section
          role="dialog"
          aria-modal="true"
          :aria-label="
            settingsGroup
              ? t('accountQuality.settings')
              : pendingRun
                ? t('accountQuality.confirmRun')
                : t('accountQuality.historyTitle')
          "
          class="quality-dialog max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-2xl bg-white p-6 shadow-2xl dark:bg-dark-800"
        >
          <template v-if="settingsGroup"
            ><h3 class="text-lg font-semibold">
              {{ t('accountQuality.settings') }} ·
              {{ groupName(settingsGroup) }}
            </h3>
            <p class="mt-2 text-xs leading-5 text-gray-500">
              {{ t('accountQuality.settingsHint') }}
            </p>
            <p class="mt-3 text-xs leading-5 text-gray-500">{{ t('accountQuality.accountCredentials') }}</p>
            <form class="mt-5 space-y-4" @submit.prevent="save">
              <label class="block text-sm"
                >{{ t('accountQuality.routingMode')
                }}<select v-model="draft.mode" class="input mt-1 w-full">
                  <option value="manual">
                    {{ t('accountQuality.mode.manual') }}
                  </option>
                  <option value="observe">
                    {{ t('accountQuality.mode.observe') }}
                  </option>
                  <option value="auto">
                    {{ t('accountQuality.mode.auto') }}
                  </option></select
                ><span class="mt-1 block text-xs leading-5 text-gray-500">{{
                  t('accountQuality.autoHint')
                }}</span></label
              ><label class="block text-sm"
                >{{ t('accountQuality.endpoint')
                }}<select v-model="draft.endpoint" class="input mt-1 w-full">
                  <option value="responses">OpenAI Responses</option>
                  <option value="chat_completions">
                    OpenAI Chat Completions
                  </option>
                  <option value="messages">Anthropic Messages</option>
                  <option value="generate_content">
                    Gemini generateContent
                  </option>
                </select></label
              >
              <fieldset
                class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
              >
                <legend class="px-2 text-sm">
                  {{ t('accountQuality.componentWeights') }}
                </legend>
                <div class="grid grid-cols-2 gap-3">
                  <label
                    v-for="key in [
                      'quality',
                      'ttft',
                      'cache',
                      'stability'
                    ] as const"
                    :key="key"
                    class="text-xs"
                    >{{ t('accountQuality.component.' + key)
                    }}<input
                      v-model.number="draft.score_weights[key]"
                      class="input mt-1 w-full"
                      type="number"
                      min="0"
                      max="100"
                      required
                  /></label>
                </div>
              </fieldset>
              <label class="flex items-center gap-2 text-sm"
                ><input v-model="draft.enabled" type="checkbox" />{{
                  t('accountQuality.enabled')
                }}</label
              ><label class="block text-sm"
                >{{ t('accountQuality.model')
                }}<input
                  v-model.trim="draft.model"
                  class="input mt-1 w-full"
                  maxlength="200"
                /><span class="mt-1 block text-xs text-gray-400">{{
                  t('accountQuality.modelHint')
                }}</span></label
              >
              <div class="grid grid-cols-2 gap-4">
                <label class="text-sm"
                  >{{ t('accountQuality.effort')
                  }}<select v-model="draft.effort" class="input mt-1 w-full">
                    <option
                      v-for="e in [
                        'unspecified',
                        'none',
                        'low',
                        'medium',
                        'high',
                        'xhigh',
                        'max',
                        'ultra'
                      ]"
                      :key="e"
                    >
                      {{ e }}
                    </option>
                  </select></label
                ><label class="text-sm"
                  >{{ t('accountQuality.tier')
                  }}<select
                    v-model="draft.service_tier"
                    class="input mt-1 w-full"
                  >
                    <option value="">{{ t('accountQuality.standard') }}</option>
                    <option value="priority">priority</option>
                    <option value="flex">flex</option>
                    <option value="ultrafast">ultrafast</option>
                  </select></label
                ><label class="text-sm"
                  >{{ t('accountQuality.intervalLabel')
                  }}<input
                    v-model.number="draft.interval_minutes"
                    class="input mt-1 w-full"
                    type="number"
                    min="5"
                    max="1440"
                    required /></label
                ><label class="text-sm"
                  >{{ t('accountQuality.budget')
                  }}<input
                    v-model.number="draft.daily_budget"
                    class="input mt-1 w-full"
                    type="number"
                    min="1"
                    max="100000"
                    required /></label
                ><label class="text-sm"
                  >{{ t('accountQuality.roundLimit')
                  }}<input
                    v-model.number="draft.max_accounts"
                    class="input mt-1 w-full"
                    type="number"
                    min="1"
                    max="200"
                    required
                /></label>
              </div>
              <div
                class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600"
              >
                <label class="block text-sm"
                  >{{ t('accountQuality.referenceModel')
                  }}<select
                    v-model="draft.reference_model"
                    class="input mt-1 w-full"
                  >
                    <option value="">
                      {{ t('accountQuality.chooseReference') }}
                    </option>
                    <option v-for="m in referenceModels" :key="m" :value="m">
                      {{ m }}
                    </option>
                  </select></label
                ><label class="flex items-center gap-2 text-sm"
                  ><input v-model="draft.trace_on_anomaly" type="checkbox" />{{
                    t('accountQuality.traceOnAnomaly')
                  }}</label
                ><label class="flex items-center gap-2 text-sm"
                  ><input v-model="draft.context_check" type="checkbox" />{{
                    t('accountQuality.contextCheck')
                  }}</label
                ><label v-if="draft.context_check" class="block text-sm"
                  >{{ t('accountQuality.contextLength')
                  }}<input
                    v-model.number="draft.context_chars"
                    class="input mt-1 w-full"
                    type="number"
                    min="4000"
                    max="128000"
                    required
                /></label>
                <p class="text-xs leading-5 text-gray-500">
                  {{ t('accountQuality.fingerprintNotice') }}
                </p>
              </div>
              <p
                class="rounded-lg bg-amber-50 p-3 text-xs leading-5 text-amber-800"
              >
                {{ t('accountQuality.runHint') }}
                <span class="mt-2 block">{{ t('accountQuality.manxueNotice') }}</span>
                <span class="mt-2 block">{{
                  t('accountQuality.budgetForecast', {
                    requests:
                      Math.ceil(1440 / Math.max(5, draft.interval_minutes)) *
                      Math.min(
                        settingsGroup?.available || 0,
                        draft.max_accounts
                      ) *
                      (4 +
                        (draft.trace_on_anomaly ? 3 : 0) +
                        (draft.context_check ? 3 : 0))
                  })
                }}</span>
              </p>
              <label v-if="draft.enabled" class="flex items-start gap-2 text-sm">
                <input v-model="draft.candy_consent_version" type="checkbox" true-value="manxue-v1" false-value="" class="mt-1" />
                {{ t('accountQuality.scheduledConsent') }}
              </label>
              <p v-if="modalError" role="alert" class="text-sm text-red-600">
                {{ modalError }}
              </p>
              <div class="flex justify-end gap-3">
                <button
                  type="button"
                  class="btn btn-secondary"
                  @click="closeModal"
                >
                  {{ t('accountQuality.cancel') }}</button
                ><button class="btn btn-primary" :disabled="saving">
                  {{
                    t(saving ? 'accountQuality.saving' : 'accountQuality.save')
                  }}
                </button>
              </div>
            </form></template
          >
          <template v-else-if="pendingRun"
            ><h3 class="text-lg font-semibold">
              {{ t('accountQuality.confirmRun') }}
            </h3>
            <p class="mt-3 text-sm leading-6 text-gray-500">
              {{ t(pendingRun.benchmark === 'candy' ? 'accountQuality.manxueNotice' : 'accountQuality.confirmHint') }}
            </p>
            <p class="mt-3 rounded-lg bg-amber-50 p-3 text-xs text-amber-800">
              {{ t('accountQuality.runHint') }} · {{ pendingRun?.benchmark }}
            </p>
            <p v-if="modalError" role="alert" class="mt-3 text-sm text-red-600">
              {{ modalError }}
            </p>
            <div class="mt-6 flex justify-end gap-3">
              <button class="btn btn-secondary" @click="closeModal">
                {{ t('accountQuality.cancel') }}</button
              ><button
                class="btn btn-primary"
                :disabled="saving"
                @click="startRun"
              >
                {{ t('accountQuality.start') }}
              </button>
            </div></template
          >
          <template v-else
            ><div class="flex justify-between">
              <h3 class="text-lg font-semibold">
                {{ t('accountQuality.historyTitle') }}
              </h3>
              <button @click="closeModal">
                {{ t('accountQuality.close') }}
              </button>
            </div>
            <p class="my-3 text-xs text-gray-500">
              {{ t('accountQuality.timingHint') }}
            </p>
            <details
              v-if="runHistory.length"
              class="my-4 rounded-xl border border-gray-200 p-3 dark:border-dark-600"
            >
              <summary class="cursor-pointer text-sm">
                {{ t('accountQuality.runHistory') }}
              </summary>
              <div
                v-for="run in runHistory"
                :key="run.id"
                class="mt-3 flex justify-between gap-3 border-t border-gray-100 pt-2 text-xs dark:border-dark-600"
              >
                <span
                  >#{{ run.id }} · {{ run.benchmark }} ·
                  {{ date(run.created_at) }}</span
                ><span
                  >{{ verdict(run.status) }} · {{ run.completed }} /
                  {{ run.planned }}</span
                >
              </div>
            </details>
            <p v-if="modalError" role="alert" class="my-3 text-sm text-red-600">
              {{ modalError }}
            </p>
            <p v-if="historyLoading" class="py-8 text-center text-gray-400">
              {{ t('accountQuality.loading') }}
            </p>
            <p
              v-else-if="!history.length && !modalError"
              class="py-8 text-center text-gray-400"
            >
              {{ t('accountQuality.noHistory') }}
            </p>
            <div
              v-for="r in history"
              :key="r.id"
              class="mt-3 rounded-xl border border-gray-200 p-4 dark:border-dark-600"
            >
              <div class="flex justify-between gap-2">
                <span class="text-xs"
                  >#{{ r.account_id }} · {{ r.model }} / {{ r.effort }}</span
                ><span
                  class="rounded-full px-2 py-1 text-xs"
                  :class="verdictClass(r.verdict)"
                  >{{ verdict(r.verdict) }}</span
                >
              </div>
              <p class="mt-2 text-[11px] text-gray-400">
                {{ r.benchmark }} · {{ date(r.created_at) }} ·
                {{ (r.duration_ms / 1000).toFixed(1) }}s
              </p>
              <p v-if="r.group_id !== historyGroup?.id" class="mt-2 text-xs text-amber-700">
                {{ t('accountQuality.evidenceSource', { group: groups.find(g => g.id === r.group_id)?.name || '#' + r.group_id }) }}
              </p>
              <p v-if="r.evidence?.provider === 'manxue' || r.benchmark === 'candy-manxue-v1'" class="mt-2 text-xs text-gray-500">
                {{ t('accountQuality.manxueResult') }}
              </p>
              <div
                v-if="r.evidence?.attribution"
                class="mt-3 rounded-lg bg-gray-50 p-3 text-xs leading-6 dark:bg-dark-900"
              >
                <p>
                  {{ t('accountQuality.referenceModel') }}:
                  {{ r.evidence.expected_model }}
                </p>
                <p>
                  {{ t('accountQuality.candidate') }}:
                  {{ r.evidence.attribution.prediction || '—' }}
                </p>
                <p>
                  {{ t('accountQuality.relativeSupport') }}:
                  {{ percent(100 * r.evidence.attribution.probability) }}
                </p>
                <p class="break-all">
                  {{ t('accountQuality.bankRevision') }}:
                  {{ r.evidence.bank_revision }}
                </p>
                <p class="text-gray-500">
                  {{ t('accountQuality.fingerprintNotice') }}
                </p>
              </div>
              <p v-if="r.detail" class="mt-2 text-xs text-red-600">
                {{ detailText(r.detail) }}
              </p>
              <details v-if="r.answer" class="mt-3 text-xs">
                <summary class="cursor-pointer text-primary-600">
                  {{ t('accountQuality.answer') }}
                </summary>
                <pre
                  class="mt-2 max-h-60 overflow-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 dark:bg-dark-900"
                  >{{ r.answer }}</pre
                >
              </details>
            </div></template
          >
        </section>
      </div></Teleport
    >
    <Teleport to="body"
      ><div
        v-if="explanation"
        class="fixed inset-0 z-[75] flex items-center justify-center bg-black/40 p-4"
        @click.self="explanation = null"
      >
        <section
          role="dialog"
          aria-modal="true"
          :aria-label="t('accountQuality.explanation')"
          class="w-full max-w-xl rounded-2xl bg-white p-6 dark:bg-dark-800"
        >
          <div class="flex justify-between gap-4">
            <h3 class="font-semibold">
              {{ explanation.name }} · {{ t('accountQuality.explanation') }}
            </h3>
            <button @click="explanation = null">
              {{ t('accountQuality.close') }}
            </button>
          </div>
          <div
            v-for="c in explanation.components || []"
            :key="c.key"
            class="mt-4 flex justify-between border-b border-gray-100 pb-3 text-sm dark:border-dark-700"
          >
            <div>
              <p>{{ t('accountQuality.component.' + c.key) }}</p>
              <p class="mt-1 text-xs text-gray-400">
                {{ c.samples }} {{ t('accountQuality.samples') }} ·
                {{ t('accountQuality.baseAndEffective') }} {{ c.weight }}% /
                {{ c.effective_weight.toFixed(1) }}%<span class="mt-1 block">{{
                  t('accountQuality.reason.' + c.key)
                }}</span>
              </p>
            </div>
            <span class="font-mono">{{
              c.score === null ? t('accountQuality.noData') : c.score.toFixed(1)
            }}</span>
          </div>
          <p class="mt-4 text-xs leading-5 text-gray-500">
            {{ t('accountQuality.metricHint') }}
          </p>
        </section>
      </div></Teleport
    >
    <Teleport to="body"
      ><div
        v-if="routingGroup"
        class="fixed inset-0 z-[76] flex items-center justify-center bg-black/40 p-4"
        @click.self="routingGroup = null"
      >
        <section
          role="dialog"
          aria-modal="true"
          :aria-label="t('accountQuality.routingDiagnostics')"
          class="quality-dialog max-h-[85vh] w-full max-w-3xl overflow-auto rounded-2xl bg-white p-6 dark:bg-dark-800"
        >
          <div class="flex justify-between gap-4">
            <h3 class="font-semibold">
              {{ routingGroup.name }} ·
              {{ t('accountQuality.routingDiagnostics') }}
            </h3>
            <button @click="routingGroup = null">
              {{ t('accountQuality.close') }}
            </button>
          </div>
          <p v-if="routingError" role="alert" class="mt-4 text-sm text-red-600">
            {{ routingError }}
          </p>
          <p v-if="routingLoading" class="py-8 text-center text-gray-500">
            {{ t('accountQuality.loading') }}
          </p>
          <template v-else-if="routingReport"
            ><p class="mt-4 rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-900">
              {{ t('accountQuality.mode.' + routingReport.configured_mode) }} ·
              {{ t('accountQuality.routeReason.' + routingReport.reason) }}
            </p>
            <p class="mt-2 text-xs text-gray-500">
              {{ routingGroup.policy.model || '—' }} /
              {{ routingGroup.policy.effort }} /
              {{
                routingGroup.policy.service_tier || t('accountQuality.standard')
              }}
              / {{ routingGroup.policy.endpoint }}
            </p>
            <p class="mt-2 text-xs leading-5 text-gray-500">
              {{ t('accountQuality.routingHint') }}
            </p>
            <div
              v-if="routingReport.applied"
              class="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-3"
            >
              <div
                v-for="(w, id) in routingReport.weights"
                :key="id"
                class="rounded-lg border border-gray-200 p-3 text-xs dark:border-dark-600"
              >
                <span>#{{ id }}</span
                ><span class="float-right font-mono text-primary-600"
                  >{{ w.multiplier.toFixed(2) }}×</span
                >
                <p v-if="w.confirmed_bad" class="mt-2 text-amber-700">
                  {{ t('accountQuality.confirmedCapabilityFailure') }}
                </p>
              </div>
            </div>
            <h4 class="mt-6 text-sm font-medium">
              {{ t('accountQuality.recentRoutes') }}
            </h4>
            <p class="mt-2 text-xs text-gray-500">
              {{ t('accountQuality.eventRetention') }}
            </p>
            <p
              v-if="!routingEvents.length"
              class="py-6 text-center text-sm text-gray-400"
            >
              {{ t('accountQuality.noRoutes') }}
            </p>
            <div
              v-for="(e, i) in routingEvents"
              :key="i"
              class="mt-3 rounded-lg border border-gray-100 p-3 text-xs dark:border-dark-600"
            >
              <div class="flex justify-between">
                <span>#{{ e.account_id }} · {{ e.scope.model }}</span
                ><span class="font-mono">{{ e.multiplier.toFixed(2) }}×</span>
              </div>
              <p class="mt-1 text-gray-500">
                {{ e.reason }} · {{ date(e.created_at) }}
              </p>
              <p class="mt-1 break-all font-mono text-gray-400">
                {{ e.request_id || '—' }}
              </p>
            </div></template
          >
        </section>
      </div></Teleport
    >
  </section>
</template>
<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { setSchedulable } from '@/api/admin/accounts'
import {
  listQualityGroups,
  qualityCatalog,
  qualityRoutingPreview,
  qualityRoutingEvents,
  type QualityRoutingPreview,
  type QualityRoutingEvent,
  type QualityBenchmark,
  type QualityAccount,
  qualityPage,
  saveQualityPolicy,
  runQuality,
  qualityRuns,
  qualityResults,
  cancelQuality,
  type QualityPolicy,
  type QualityGroup,
  type QualityPage,
  type QualityRun,
  type QualityResult
} from '@/api/admin/accountQuality'
const { t, locale } = useI18n()
function isAccountUnavailable(account: QualityAccount): boolean {
  return account.status !== 'active' || !account.schedulable ||
    [account.temp_unschedulable_until, account.rate_limit_reset_at, account.overload_until]
      .some(until => !!until && new Date(until).getTime() > Date.now())
}
function accountHomepageUrl(account: QualityAccount): string {
  if (account.type !== 'apikey' || typeof account.homepage_url !== 'string') return ''
  const url = sanitizeUrl(account.homepage_url)
  return url ? new URL(url).origin : ''
}

const props = withDefaults(
  defineProps<{ selectedIds?: number[]; refreshToken?: number; pauseAutoRefresh?: boolean }>(),
  {
    selectedIds: () => []
  }
)
const selectionMetadata = new Map<number, QualityAccount>()
const emit = defineEmits<{
  selection: [rows: QualityAccount[]]
  create: []
  edit: [id: number]
  test: [id: number]
  'show-temp-unsched': [id: number]
  schedulingChange: [id: number, schedulable: boolean]
}>()
const groups = ref<QualityGroup[]>([]),
  pages = reactive<Record<number, QualityPage>>({}),
  busy = reactive<Record<number, boolean>>({}),
  rowErrors = reactive<Record<number, string>>({}),
  collapsed = reactive<Record<number, boolean>>({})
const scheduling = reactive<Record<number, boolean>>({}),
  schedulingErrors = reactive<Record<number, string>>({})
const loading = ref(false),
  error = ref(''),
  notice = ref(''),
  search = ref(''),
  metricsModel = ref(''),
  platform = ref(''),
  status = ref(''),
  sort = ref('score'),
  benchmark = ref<QualityBenchmark>('candy'),
  referenceModels = ref<string[]>([]),
  explanation = ref<QualityAccount | null>(null),
  saving = ref(false),
  modalError = ref('')
const routingGroup = ref<QualityGroup | null>(null),
  routingReport = ref<QualityRoutingPreview | null>(null),
  routingEvents = ref<QualityRoutingEvent[]>([]),
  routingLoading = ref(false),
  routingError = ref('')
const settingsGroup = ref<QualityGroup | null>(null),
  historyGroup = ref<QualityGroup | null>(null),
  runHistory = ref<QualityRun[]>([]),
  history = ref<QualityResult[]>([]),
  historyLoading = ref(false),
  pendingRun = ref<{
    group: QualityGroup
    id: number
    benchmark: QualityBenchmark
  } | null>(null),
  activeRun = ref<QualityRun | null>(null)
let historyRequestVersion = 0
const defaults: QualityPolicy = {
  enabled: false,
  candy_consent_version: '',
  interval_minutes: 15,
  model: '',
  effort: 'low',
  service_tier: '',
  daily_budget: 200,
  max_accounts: 50,
  mode: 'observe',
  endpoint: 'responses',
  trace_on_anomaly: false,
  reference_model: '',
  context_check: false,
  context_chars: 32000,
  score_weights: { quality: 40, ttft: 25, cache: 15, stability: 20 }
}
const draft = reactive<QualityPolicy>({ ...defaults }),
  controllers = new Map<number, AbortController>()
let timer: ReturnType<typeof setInterval> | undefined,
  debounce: ReturnType<typeof setTimeout> | undefined,
  disposed = false
let pageGeneration = 0,
  refreshingSilently = false,
  polling = false
const {
  enabled: autoRefreshEnabled,
  intervalSeconds: autoRefreshIntervalSeconds,
  countdown: autoRefreshCountdown,
  fetching: autoRefreshFetching,
  intervals: autoRefreshIntervals,
  setEnabled: setAutoRefreshEnabled,
  setInterval: setAutoRefreshInterval,
  resetCountdown: resetAutoRefreshCountdown,
  start: startAutoRefresh
} = useAutoRefresh({
  storageKey: 'account-quality-auto-refresh',
  defaultEnabled: true,
  defaultInterval: 10,
  onRefresh: refreshVisibleGroups,
  shouldPause: backgroundRefreshPaused
})
function changeAutoRefresh(event: Event) {
  const seconds = Number((event.target as HTMLSelectElement).value)
  if (seconds === 0) setAutoRefreshEnabled(false)
  else {
    setAutoRefreshInterval(seconds)
    setAutoRefreshEnabled(true)
  }
}
function backgroundRefreshPaused() {
  return disposed || document.hidden || props.pauseAutoRefresh || loading.value ||
    refreshingSilently || saving.value || !!debounce ||
    !!(settingsGroup.value || historyGroup.value || routingGroup.value || explanation.value || pendingRun.value) ||
    Object.values(busy).some(Boolean) || Object.values(scheduling).some(Boolean)
}
const platforms = computed(() => [
  ...new Set(groups.value.map((g) => g.platform))
])
const visibleGroups = computed(() =>
  groups.value.filter(
    (g) =>
      (!platform.value || g.platform === platform.value) &&
      (!search.value ||
        groupName(g).toLowerCase().includes(search.value.toLowerCase()) ||
        (pages[g.id]?.total || 0) > 0)
  )
)
const overview = computed(() => [
  { label: t('accountQuality.groups'), value: groups.value.length, icon: 'grid' as const, tone: 'quality-stat-icon--neutral' },
  {
    label: t('accountQuality.memberships'),
    value: groups.value.reduce((n, g) => n + g.available, 0),
    icon: 'checkCircle' as const,
    tone: 'quality-stat-icon--green'
  },
  {
    label: t('accountQuality.interval'),
    value: '15 ' + t('accountQuality.minutes'),
    icon: 'clock' as const,
    tone: 'quality-stat-icon--neutral'
  }
])
const groupName = (g: QualityGroup) =>
  g.id ? g.name : t('accountQuality.ungrouped')
const date = (v: string | null) =>
  v
    ? new Intl.DateTimeFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
      }).format(new Date(v))
    : '—'
const latency = (v: number | null) =>
  v === null ? '—' : (v / 1000).toFixed(2) + 's'
const percent = (v: number | null) =>
  v == null || !Number.isFinite(v) ? '—' : v.toFixed(1) + '%'
const verdict = (v: string) => t('accountQuality.' + v)
const verdictClass = (v: string) =>
  v === 'passed'
    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
    : v === 'incorrect'
      ? 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
      : v === 'error' || v === 'unparseable' || v === 'inconsistent'
        ? 'bg-amber-50 text-amber-700'
        : 'bg-gray-100 text-gray-500 dark:bg-dark-700'
function validCandyPolicy(p: QualityPolicy) {
  return p.endpoint === 'responses' &&
    ['low', 'medium', 'high', 'xhigh', 'max', 'ultra'].includes(p.effort) &&
    ['', 'priority', 'ultrafast'].includes(p.service_tier)
}
function candyAccountSupported(a: QualityAccount) {
  return a.type === 'apikey' && ['openai', 'kimi', 'zhipu', 'deepseek', 'opencode_go'].includes(a.platform)
}
function canInspect(a: QualityAccount, p: QualityPolicy) {
  if (benchmark.value === 'candy') return candyAccountSupported(a) && validCandyPolicy(p)
  switch (p.endpoint) {
    case 'responses': return a.platform === 'openai' || a.type === 'apikey'
    case 'chat_completions': return a.type === 'apikey'
    case 'messages': return a.platform === 'anthropic' || a.type === 'apikey'
    case 'generate_content': return a.platform === 'gemini'
    default: return false
  }
}
function inspectionHint(a: QualityAccount, p: QualityPolicy) {
  if (benchmark.value === 'candy' && a.type === 'oauth') return t('accountQuality.oauthUnsupported')
  if (benchmark.value === 'candy' && !validCandyPolicy(p)) return t('accountQuality.manxueParameters')
  return canInspect(a, p) ? '' : t('accountQuality.failure.unsupported')
}
function detailText(value: string) {
  if (value.startsWith('reused_result:'))
    return t('accountQuality.reusedEvidence', { id: value.split(':')[1] })
  const keys: Record<string, string> = {
    authentication_failed: 'auth',
    upstream_balance_exhausted: 'balance',
    access_denied: 'access',
    rate_limited: 'rate',
    timeout: 'timeout',
    cancelled: 'cancelled',
    request_rejected: 'request',
    upstream_server_error: 'server',
    upstream_or_transport_error: 'transport',
    incomplete_response: 'incomplete',
    insufficient_valid_fingerprint_outputs: 'fingerprint',
    account_not_schedulable: 'unavailable',
    unsupported_account: 'unsupported',
    use_unspecified_effort_for_native_gemini: 'unsupported',
    response_too_large: 'large',
    manxue_unsupported_account: 'manxueAccount',
    manxue_public_https_required: 'manxueURL',
    manxue_unsupported_parameters: 'manxueParameters',
    manxue_invalid_response: 'manxueResponse',
    manxue_evaluation_error: 'manxueEvaluation',
    manxue_task_failed: 'manxueEvaluation',
    manxue_request_failed: 'manxueTransport',
    manxue_transport_error: 'manxueTransport',
    account_unavailable: 'unavailable',
    missing_api_key: 'auth'
  }
  if (value.startsWith('manxue_http_'))
    return t('accountQuality.manxueHTTP', { status: value.slice('manxue_http_'.length) })
  return keys[value] ? t('accountQuality.failure.' + keys[value]) : value
}
function message(e: unknown) {
  return e instanceof Error ? e.message : t('accountQuality.failed')
}
async function load(g: QualityGroup, page = 1, silent = false) {
  if (disposed) return
  if (silent && controllers.has(g.id)) return
  controllers.get(g.id)?.abort()
  const c = new AbortController()
  controllers.set(g.id, c)
  if (!silent) {
    busy[g.id] = true
    rowErrors[g.id] = ''
  }
  try {
    const q = groupName(g).toLowerCase().includes(search.value.toLowerCase())
      ? ''
      : search.value
    const data = await qualityPage(
      g.id,
      { page, page_size: 8, search: q, status: status.value, sort: sort.value, metrics_model: metricsModel.value },
      c.signal
    )
    if (!disposed && !c.signal.aborted) {
      pages[g.id] = data
      rowErrors[g.id] = ''
      for (const row of data.items) selectionMetadata.set(row.id, row)
    }
  } catch (e) {
    if (!c.signal.aborted) rowErrors[g.id] = message(e)
  } finally {
    if (controllers.get(g.id) === c) {
      busy[g.id] = false
      controllers.delete(g.id)
    }
  }
}
function invalidatePages() {
  pageGeneration++
  controllers.forEach(c => c.abort())
  for (const id of Object.keys(pages)) delete pages[Number(id)]
}
async function refresh() {
  if (disposed) return
  invalidatePages()
  loading.value = true
  error.value = ''
  try {
    const data = await listQualityGroups()
    if (disposed) return
    groups.value = data.sort((a, b) => b.total - a.total || a.id - b.id)
    for (let i = 0; i < groups.value.length; i++) {
      const g = groups.value[i]
      if (!(g.id in collapsed)) collapsed[g.id] = i >= 3
    }
    await Promise.all(
      groups.value
        .filter((g) => !collapsed[g.id])
        .map((g) => load(g, pages[g.id]?.page || 1))
    )
    if (!disposed) await poll(false)
  } catch (e) {
    error.value = message(e)
  } finally {
    loading.value = false
  }
}
async function refreshVisibleGroups() {
  if (backgroundRefreshPaused()) return
  refreshingSilently = true
  const generation = pageGeneration
  try {
    const data = await listQualityGroups()
    if (disposed || generation !== pageGeneration) return
    // Keep existing cards in place while updating counts and policy summaries.
    const order = new Map(groups.value.map((g, i) => [g.id, i]))
    groups.value = data.sort((a, b) =>
      (order.get(a.id) ?? Infinity) - (order.get(b.id) ?? Infinity) || b.total - a.total || a.id - b.id
    )
    for (let i = 0; i < groups.value.length; i++) {
      const g = groups.value[i]
      if (!(g.id in collapsed)) collapsed[g.id] = i >= 3
    }
    error.value = ''
    await Promise.all(groups.value
      .filter(g => !collapsed[g.id] && (!platform.value || g.platform === platform.value))
      .map(g => load(g, pages[g.id]?.page || 1, true)))
  } catch (e) {
    if (!disposed && generation === pageGeneration) error.value = message(e)
  } finally {
    refreshingSilently = false
  }
}
function handleVisibilityChange() {
  if (!autoRefreshEnabled.value || backgroundRefreshPaused()) return
  resetAutoRefreshCountdown()
  void refreshVisibleGroups()
  void poll().catch(e => { if (!disposed) error.value = message(e) })
}
async function toggleScheduling(account: QualityAccount) {
  if (scheduling[account.id]) return
  scheduling[account.id] = true
  schedulingErrors[account.id] = ''
  let saved = false
  try {
    const updated = await setSchedulable(account.id, !account.schedulable)
    saved = true
    const affectedGroups = groups.value.filter(g => pages[g.id]?.items.some(row => row.id === account.id))
    for (const g of affectedGroups) {
      if (collapsed[g.id]) delete pages[g.id]
    }
    for (const page of Object.values(pages)) {
      for (const row of page.items) {
        if (row.id === account.id) row.schedulable = updated.schedulable
      }
    }
    const selected = selectionMetadata.get(account.id)
    if (selected) selected.schedulable = updated.schedulable
    emit('schedulingChange', account.id, updated.schedulable)
    await Promise.all([
      listQualityGroups().then(data => { groups.value = data.sort((a,b) => b.total-a.total || a.id-b.id) }),
      ...affectedGroups.filter(g => !collapsed[g.id]).map(g => load(g, pages[g.id]?.page || 1))
    ])
  } catch (e) {
    if (saved) error.value = message(e)
    else schedulingErrors[account.id] = t('admin.accounts.failedToToggleSchedulable')
  } finally {
    scheduling[account.id] = false
  }
}
function emitSelection(ids: Set<number>) {
  emit(
    'selection',
    [...ids]
      .map((id) => selectionMetadata.get(id))
      .filter((row): row is QualityAccount => !!row)
  )
}
function toggleSelection(row: QualityAccount) {
  selectionMetadata.set(row.id, row)
  const ids = new Set(props.selectedIds)
  if (ids.has(row.id)) ids.delete(row.id)
  else ids.add(row.id)
  emitSelection(ids)
}
function isGroupPageSelected(id: number) {
  const rows = pages[id]?.items || []
  return (
    rows.length > 0 && rows.every((row) => props.selectedIds.includes(row.id))
  )
}
function toggleGroupSelection(id: number) {
  const rows = pages[id]?.items || []
  const remove = isGroupPageSelected(id)
  const ids = new Set(props.selectedIds)
  for (const row of rows) {
    selectionMetadata.set(row.id, row)
    if (remove) ids.delete(row.id)
    else ids.add(row.id)
  }
  emitSelection(ids)
}
function toggle(g: QualityGroup) {
  collapsed[g.id] = !collapsed[g.id]
  if (!collapsed[g.id] && !pages[g.id]) void load(g)
}
function openSettings(g: QualityGroup) {
  settingsGroup.value = g
  Object.assign(draft, JSON.parse(JSON.stringify({ ...defaults, ...g.policy })))
  modalError.value = ''
}
function closeModal() {
  if (saving.value) return
  historyRequestVersion++
  settingsGroup.value = null
  historyGroup.value = null
  pendingRun.value = null
  modalError.value = ''
}
async function save() {
  if (!settingsGroup.value) return
  saving.value = true
  modalError.value = ''
  try {
    if (!draft.reference_model && referenceModels.value.includes(draft.model)) {
      draft.reference_model = draft.model
    }
    if (draft.enabled && !validCandyPolicy(draft)) {
      modalError.value = t('accountQuality.manxueParameters')
      return
    }
    if (draft.enabled && draft.candy_consent_version !== 'manxue-v1') {
      modalError.value = t('accountQuality.consentRequired')
      return
    }
    await saveQualityPolicy(settingsGroup.value.id, { ...draft })
    saving.value = false
    closeModal()
    notice.value = t('accountQuality.saved')
    await refresh()
  } catch (e) {
    modalError.value = message(e)
  } finally {
    saving.value = false
  }
}
function requestRun(g: QualityGroup, id = 0) {
  notice.value = ''
  if (!g.policy.model) {
    openSettings(g)
    modalError.value = t('accountQuality.configure')
    return
  }
  if (benchmark.value === 'candy' && !validCandyPolicy(g.policy)) {
    openSettings(g)
    modalError.value = t('accountQuality.manxueParameters')
    return
  }
  if (benchmark.value === 'modeltrace' && !g.policy.reference_model) {
    openSettings(g)
    modalError.value = t('accountQuality.referenceRequired')
    return
  }
  pendingRun.value = { group: g, id, benchmark: benchmark.value }
  modalError.value = ''
}
async function startRun() {
  if (!pendingRun.value) return
  saving.value = true
  modalError.value = ''
  try {
    activeRun.value = await runQuality(
      pendingRun.value.group.id,
      pendingRun.value.id,
      pendingRun.value.benchmark
    )
    saving.value = false
    closeModal()
    notice.value = t('accountQuality.runQueued')
  } catch (e) {
    modalError.value = message(e)
  } finally {
    saving.value = false
  }
}
async function poll(refreshCompleted = true) {
  if (disposed || polling) return
  polling = true
  try {
    const runs = await qualityRuns(0)
    if (disposed) return
    const run = runs.find((r) => r.status === 'running' || r.status === 'queued')
    if (run) {
      activeRun.value = run
      return
    }
    if (activeRun.value) {
      activeRun.value = null
      if (refreshCompleted) await refreshVisibleGroups()
    }
  } finally {
    polling = false
  }
}
async function cancelRound() {
  if (!activeRun.value) return
  try {
    await cancelQuality(activeRun.value.id)
    await poll()
  } catch (e) {
    error.value = message(e)
  }
}
async function openRouting(g: QualityGroup) {
  routingGroup.value = g
  routingReport.value = null
  routingEvents.value = []
  routingError.value = ''
  routingLoading.value = true
  try {
    const [report, events] = await Promise.all([
      qualityRoutingPreview(g),
      qualityRoutingEvents(g.id)
    ])
    if (routingGroup.value?.id === g.id) {
      routingReport.value = report
      routingEvents.value = events
    }
  } catch (e) {
    routingError.value = message(e)
  } finally {
    routingLoading.value = false
  }
}
async function openHistory(g: QualityGroup, id: number, resultIds?: number[]) {
  const requestVersion = ++historyRequestVersion
  historyGroup.value = g
  modalError.value = ''
  history.value = []
  runHistory.value = []
  historyLoading.value = true
  try {
    const [results, runs] = await Promise.all([
      qualityResults(g.id, id, resultIds),
      qualityRuns(g.id)
    ])
    if (requestVersion !== historyRequestVersion) return
    history.value = results
    runHistory.value = runs
  } catch (e) {
    modalError.value = message(e)
  } finally {
    if (requestVersion === historyRequestVersion) historyLoading.value = false
  }
}
watch(
  () => props.refreshToken,
  () => {
    void refresh()
  }
)
watch([search, sort, status, metricsModel], () => {
  invalidatePages()
  if (debounce) clearTimeout(debounce)
  debounce = setTimeout(() => {
    debounce = undefined
    for (const g of groups.value)
      if (search.value || !collapsed[g.id]) void load(g)
  }, 300)
})
onMounted(async () => {
  try {
    const catalog = await qualityCatalog()
    if (disposed) return
    referenceModels.value = catalog.reference_models
  } catch {
    if (!disposed) error.value = t('accountQuality.failed')
  }
  if (disposed) return
  await refresh()
  if (disposed) return
  if (autoRefreshEnabled.value) {
    resetAutoRefreshCountdown()
    startAutoRefresh()
  }
  document.addEventListener('visibilitychange', handleVisibilityChange)
  timer = setInterval(() => {
    if (backgroundRefreshPaused()) return
    void poll().catch((e) => {
      if (!disposed) error.value = message(e)
    })
  }, 5000)
})
onUnmounted(() => {
  disposed = true
  historyRequestVersion++
  if (timer) clearInterval(timer)
  if (debounce) clearTimeout(debounce)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  controllers.forEach((c) => c.abort())
})
</script>

<style scoped>
.quality-workspace {
  @apply text-gray-700 dark:text-gray-200;
}

.quality-header {
  @apply mb-5 flex flex-wrap items-center justify-between gap-4;
}

.quality-title {
  @apply text-[22px] font-semibold leading-8 tracking-tight text-gray-900 dark:text-white;
}

.quality-subtitle {
  @apply mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400;
}

.quality-header-actions {
  @apply flex flex-wrap items-center gap-2;
}

.quality-model-filter {
  @apply w-56;
}

.quality-auto-refresh {
  @apply inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400;
}

.quality-auto-refresh select {
  @apply h-full cursor-pointer border-0 bg-transparent py-0 pl-0 pr-5 text-xs text-gray-600 focus:ring-0 dark:text-gray-300;
}

.quality-button {
  @apply inline-flex h-9 shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-lg border border-transparent px-3 text-xs font-medium leading-none transition-colors duration-150;
  @apply focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-800;
  @apply disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-40;
}

.quality-button--primary {
  @apply border-primary-600 bg-primary-600 text-white shadow-sm hover:border-primary-700 hover:bg-primary-700 dark:border-primary-500 dark:bg-primary-500 dark:text-dark-950 dark:hover:bg-primary-400;
}

.quality-button--secondary {
  @apply border-gray-200 bg-white text-gray-600 shadow-sm hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-dark-500 dark:hover:bg-dark-700 dark:hover:text-white;
}

.quality-button--soft {
  @apply border-primary-100 bg-primary-50 text-primary-700 hover:border-primary-200 hover:bg-primary-100 dark:border-primary-800/50 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:bg-primary-900/40;
}

.quality-button--ghost {
  @apply text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-gray-100;
}

.quality-button--small {
  @apply h-8 gap-1.5 rounded-md px-2.5 text-xs;
}

.quality-button--icon {
  @apply w-8 px-0;
}

.quality-overview {
  @apply mb-4 grid grid-cols-3 overflow-hidden rounded-xl border border-gray-200/80 bg-white dark:border-dark-700 dark:bg-dark-800;
}

.quality-stat {
  @apply flex items-center gap-3 px-5 py-4;
}

.quality-stat + .quality-stat {
  @apply border-l border-gray-100 dark:border-dark-700;
}

.quality-stat-icon {
  @apply flex h-10 w-10 shrink-0 items-center justify-center rounded-xl;
}

.quality-stat-icon--neutral {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400;
}

.quality-stat-icon--green {
  @apply bg-primary-50 text-primary-600 dark:bg-primary-900/25 dark:text-primary-400;
}

.quality-stat-label {
  @apply text-xs leading-5 text-gray-500 dark:text-gray-400;
}

.quality-stat-value {
  @apply text-xl font-semibold leading-7 tracking-tight text-gray-900 tabular-nums dark:text-white;
}

.quality-toolbar {
  @apply flex flex-wrap items-center gap-2 rounded-xl border border-gray-200/80 bg-white p-3 dark:border-dark-700 dark:bg-dark-800;
}

.quality-search {
  @apply flex h-9 min-w-[180px] flex-[1_1_220px] items-center gap-2 rounded-lg border border-gray-200 bg-gray-50/70 px-3 text-gray-400 transition-colors focus-within:border-primary-400 focus-within:ring-2 focus-within:ring-primary-500/10 dark:border-dark-600 dark:bg-dark-900/40;
}

.quality-search input {
  @apply min-w-0 flex-1 border-0 bg-transparent p-0 text-xs leading-5 text-gray-900 outline-none placeholder:text-gray-400 focus:ring-0 dark:text-gray-100;
}

.quality-select {
  @apply h-9 min-w-0 cursor-pointer rounded-lg border border-gray-200 bg-white py-0 pl-3 pr-7 text-xs text-gray-600 transition-colors hover:border-gray-300 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500/10 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300;
}

.quality-benchmark {
  @apply bg-gray-50/70 font-medium text-gray-700 dark:bg-dark-900/40 dark:text-gray-200;
}

.quality-context {
  @apply flex flex-wrap items-center gap-x-1.5 gap-y-1 px-1 py-3 text-[11px] leading-5 text-gray-400 dark:text-gray-500;
}

.quality-context-label {
  @apply inline-flex items-center gap-1.5 text-gray-500 dark:text-gray-400;
}

.quality-context-divider {
  @apply mx-2 h-3 w-px bg-gray-200 dark:bg-dark-600;
}

.quality-groups {
  @apply space-y-4;
}

.quality-group {
  @apply overflow-hidden rounded-xl border border-gray-200/80 bg-white dark:border-dark-700 dark:bg-dark-800;
  box-shadow: 0 2px 5px -3px rgb(15 23 42 / 0.08);
}

.quality-group-header {
  @apply flex flex-wrap items-center gap-3 px-4 py-3.5;
}

.quality-group-title {
  @apply flex min-w-0 flex-1 items-center gap-2.5 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500;
}

.quality-group-icon {
  @apply flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-gray-100 bg-gray-50 text-gray-400 dark:border-dark-600 dark:bg-dark-700/60;
}

.quality-group-name {
  @apply block max-w-[300px] truncate text-sm font-semibold leading-5 text-gray-900 dark:text-white;
}

.quality-group-meta {
  @apply mt-1 flex flex-wrap items-center gap-2 text-[11px] leading-4 text-gray-400;
}

.quality-platform {
  @apply rounded border border-gray-200/70 bg-gray-50 px-1.5 text-[10px] leading-4 text-gray-500 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-400;
}

.quality-available {
  @apply inline-flex items-center gap-1 text-emerald-600 dark:text-emerald-400;
}

.quality-status-dot {
  @apply h-1 w-1 rounded-full bg-emerald-500;
}

.quality-group-actions {
  @apply ml-auto flex flex-wrap items-center gap-1;
}

.quality-schedule {
  @apply mr-2 inline-flex items-center gap-1.5 text-[11px] text-gray-400;
}

.quality-schedule--active {
  @apply text-primary-600 dark:text-primary-400;
}

.quality-action-divider {
  @apply mx-1 h-4 w-px bg-gray-200 dark:bg-dark-600;
}

.quality-group-context {
  @apply flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-y border-gray-100 bg-gray-50/70 px-4 py-2 text-[11px] leading-5 text-gray-500 dark:border-dark-700 dark:bg-dark-900/30 dark:text-gray-400;
}

.quality-table-scroll {
  @apply overflow-x-auto;
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
}

.quality-table {
  @apply w-full whitespace-nowrap text-left text-xs;
}

.quality-table th {
  @apply bg-gray-50/50 px-3 py-2.5 text-[11px] font-medium tracking-normal text-gray-400 dark:bg-dark-900/20 dark:text-gray-500;
  text-transform: none;
}

.quality-table td {
  @apply px-3 py-3 align-middle;
}

.quality-table th:first-child,
.quality-table td:first-child {
  @apply min-w-[200px] pl-4;
}

.quality-table th:last-child,
.quality-table td:last-child {
  @apply sticky right-0 pl-3 pr-4;
  box-shadow: -6px 0 8px -8px rgb(15 23 42 / 0.2);
}

.quality-table th:last-child {
  @apply bg-gray-50 dark:bg-dark-800;
}

.quality-table td:last-child {
  @apply bg-white dark:bg-dark-800;
}

.quality-table tbody tr:hover td {
  @apply bg-gray-50 dark:bg-dark-700;
}

.quality-table tbody tr.quality-row--selected td {
  @apply bg-primary-50 dark:bg-dark-700;
}

.quality-account-link,
.quality-account-name {
  @apply block max-w-[200px] truncate text-xs font-medium text-gray-800 dark:text-gray-100;
}

.quality-account-link {
  @apply decoration-primary-300 underline-offset-4 transition-colors hover:text-primary-600 hover:underline dark:hover:text-primary-400;
}

.quality-account-identity--error .quality-account-link,
.quality-account-identity--error .quality-account-name {
  @apply text-red-600 decoration-red-300 dark:text-red-400 dark:decoration-red-700;
}

.quality-account-identity--disabled .quality-account-link,
.quality-account-identity--disabled .quality-account-name {
  @apply text-yellow-600 decoration-yellow-300 dark:text-yellow-400 dark:decoration-yellow-700;
}

.quality-account-meta {
  @apply mt-1 pl-[38px] text-[10px] leading-4 text-gray-400;
}

.quality-verdict {
  @apply inline-flex items-center rounded-md px-2 py-1 text-[11px] font-medium leading-4 ring-1 ring-inset ring-gray-900/5 transition-opacity hover:opacity-75 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:ring-white/10;
}

.quality-score {
  @apply inline-flex min-w-[46px] items-center justify-center rounded-md bg-primary-50 px-2 py-1 font-mono text-xs font-semibold text-primary-700 transition-colors hover:bg-primary-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:bg-primary-900/25 dark:text-primary-300 dark:hover:bg-primary-900/40;
}

.quality-row-actions {
  @apply flex items-center justify-end gap-0.5;
}

.quality-row-actions .quality-button--soft {
  @apply mx-1;
}

.quality-row-hint {
  @apply mt-1 flex items-center justify-end gap-1 text-[10px] leading-4 text-gray-400;
}

.quality-group-footer {
  @apply flex items-center justify-between gap-3 border-t border-gray-100 px-4 py-2.5 text-[11px] text-gray-400 dark:border-dark-700;
}

.quality-pagination {
  @apply flex items-center gap-3 tabular-nums;
}

.quality-pagination .quality-button {
  @apply h-7 px-2 text-[11px] shadow-none;
}

.quality-workspace :deep(input[type='checkbox']) {
  @apply h-3.5 w-3.5 shrink-0 cursor-pointer rounded border-gray-300 accent-primary-600 disabled:cursor-not-allowed disabled:opacity-40;
}

.quality-dialog .btn {
  @apply h-9 rounded-lg px-3 py-0 text-xs shadow-none;
  background-image: none;
}

.quality-dialog .btn-primary {
  @apply bg-primary-600 hover:bg-primary-700;
}

.quality-dialog .input {
  @apply h-10 rounded-lg px-3 py-2 text-sm;
}

.quality-dialog label {
  @apply text-gray-600 dark:text-gray-300;
}

@media (max-width: 1023px) {
  .quality-group-actions {
    @apply w-full border-t border-gray-100 pt-2.5 dark:border-dark-700;
  }

  .quality-schedule {
    @apply mr-auto;
  }
}

@media (max-width: 639px) {
  .quality-header-actions {
    @apply w-full;
  }

  .quality-stat {
    @apply gap-2 px-3 py-3;
  }

  .quality-stat-icon {
    @apply hidden;
  }

  .quality-stat-label {
    @apply text-[10px];
  }

  .quality-stat-value {
    @apply text-lg;
  }

  .quality-toolbar {
    @apply gap-2 p-2.5;
  }

  .quality-search {
    @apply basis-full;
  }

  .quality-select {
    @apply flex-1 basis-[calc(50%-4px)];
  }

  .quality-context-divider {
    @apply hidden;
  }

  .quality-schedule {
    @apply mb-1 w-full;
  }

  .quality-group-name {
    @apply max-w-[220px];
  }

  .quality-group-header,
  .quality-group-context,
  .quality-group-footer {
    @apply px-3;
  }

  .quality-pagination {
    @apply gap-2;
  }
}
</style>
