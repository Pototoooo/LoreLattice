import assert from 'node:assert/strict'
import test from 'node:test'

import { baseCompile } from '@intlify/message-compiler'

import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'
import ruRU from './locales/ru-RU.ts'
import zhCN from './locales/zh-CN.ts'

interface LocaleTree {
  [key: string]: string | LocaleTree
}

function collectMessageCompileErrors(
  value: string | LocaleTree,
  path: string,
  errors: string[],
): void {
  if (typeof value === 'string') {
    baseCompile(value, {
      onError(error) {
        errors.push(`${path}: [${error.code}] ${error.message}`)
      },
    })
    return
  }

  for (const [key, child] of Object.entries(value)) {
    collectMessageCompileErrors(child, `${path}.${key}`, errors)
  }
}

test('all locale messages compile without vue-i18n syntax errors', () => {
  const errors: string[] = []
  const locales = { 'zh-CN': zhCN, 'en-US': enUS, 'ru-RU': ruRU, 'ko-KR': koKR }

  for (const [locale, messages] of Object.entries(locales)) {
    collectMessageCompileErrors(messages as LocaleTree, locale, errors)
  }

  assert.deepEqual(errors, [])
})
