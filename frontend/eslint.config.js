import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import tsPlugin from '@typescript-eslint/eslint-plugin'
import tsParser from '@typescript-eslint/parser'

export default [
  { ignores: ['dist', 'coverage'] },
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: { parser: tsParser, parserOptions: { ecmaVersion: 2022, sourceType: 'module' }, globals: globals.browser },
    plugins: { '@typescript-eslint': tsPlugin, 'react-hooks': reactHooks },
    rules: { ...tsPlugin.configs.recommended.rules, ...reactHooks.configs.recommended.rules },
  },
]
