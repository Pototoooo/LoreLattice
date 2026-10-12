import assert from 'node:assert/strict'
import test from 'node:test'

import { isQuickAnswerAgentId, isAgentStreamAgentId, reconcileBuiltinAgentMode } from './agent-mode.ts'

test('isQuickAnswerAgentId treats builtin quick-answer as RAG mode', () => {
  assert.equal(isQuickAnswerAgentId('builtin-quick-answer'), true)
  assert.equal(isQuickAnswerAgentId(undefined), true)
  assert.equal(isQuickAnswerAgentId('builtin-smart-reasoning'), false)
})

test('isAgentStreamAgentId prefers selectedAgentId for builtins', () => {
  assert.equal(
    isAgentStreamAgentId('builtin-quick-answer', true),
    false,
    'stale isAgentEnabled must not flip quick-answer into agent stream',
  )
  assert.equal(isAgentStreamAgentId('builtin-smart-reasoning', false), true)
  assert.equal(isAgentStreamAgentId('custom-agent', true), true)
  assert.equal(isAgentStreamAgentId('custom-agent', false), false)
})

test('reconcileBuiltinAgentMode repairs drifted localStorage flags', () => {
  const quick = { selectedAgentId: 'builtin-quick-answer', isAgentEnabled: true }
  assert.equal(reconcileBuiltinAgentMode(quick), true)
  assert.equal(quick.isAgentEnabled, false)

  const reasoning = { selectedAgentId: 'builtin-smart-reasoning', isAgentEnabled: false }
  assert.equal(reconcileBuiltinAgentMode(reasoning), true)
  assert.equal(reasoning.isAgentEnabled, true)

  const custom = { selectedAgentId: 'custom-agent', isAgentEnabled: true }
  assert.equal(reconcileBuiltinAgentMode(custom), false)
})
