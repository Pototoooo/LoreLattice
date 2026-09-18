import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const read = (url) => readFileSync(url, 'utf8')

const theme = read(new URL('./ui-refresh.less', import.meta.url))
const main = read(new URL('../../main.ts', import.meta.url))
const platform = read(new URL('../../views/platform/index.vue', import.meta.url))

test('complete UI refresh is loaded before the app mounts', () => {
  assert.match(main, /import "@\/assets\/theme\/ui-refresh\.less"/)
  assert.match(main, /document\.documentElement\.classList\.add\("ll-ui-refresh"\)/)
  assert.match(theme, /^html\.ll-ui-refresh \{/m)
})

test('Figma desktop geometry and reusable component states stay represented', () => {
  assert.match(theme, /width: 260px !important/)
  assert.match(theme, /height: 68px/)
  assert.match(theme, /height: 216px !important/)
  assert.match(theme, /--ll-control-height: 44px/)
  assert.match(theme, /\.t-button/)
  assert.match(theme, /\.t-dialog/)
  assert.match(theme, /\.t-table/)
  assert.match(theme, /\.t-tabs__nav-item/)
})

test('all final Figma product modules have a styled production surface', () => {
  const moduleSelectors = [
    // Knowledge, documents, Wiki and upload flows.
    '.kb-list-content',
    '.knowledge-layout',
    '.doc-list-row',
    '.wiki-sidebar',
    '.faq-entry-card',
    // Agent, chat and session flows.
    '.agent-list-content',
    '.settings-overlay',
    '.dialogue-wrap',
    '.dialogue-answers',
    '.input-container',
    // Spaces, members and sharing.
    '.org-list-content',
    '.settings-content .member-card',
    '.settings-content .invite-card',
    '.t-dialog',
    // Models, data engines, settings and system administration.
    '.settings-content .model-card',
    '.settings-overlay',
    '.settings-content .engine-card',
    '.system-settings',
    // Auth, onboarding, embed and integrations.
    '.login-layout',
    '.workspace-onboarding',
    '.embed-page',
    '.integration-layout',
  ]

  for (const selector of moduleSelectors) {
    assert.ok(theme.includes(selector), `missing refreshed surface: ${selector}`)
  }
})

test('platform layout provides a stable app-level header around every route', () => {
  assert.match(platform, /class="platform-shell"/)
  assert.match(platform, /class="platform-topbar"/)
  assert.match(platform, /class="platform-route-outlet"/)
  assert.match(platform, /const pageTitle = computed/)
})
