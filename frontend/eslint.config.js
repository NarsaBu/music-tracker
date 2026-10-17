import js from '@eslint/js';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist', 'storybook-static', 'coverage'] },
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: { ecmaVersion: 2023, globals: globals.browser },
    plugins: { 'react-hooks': reactHooks, 'react-refresh': reactRefresh },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      // MVC: Views and Controllers never talk to the network directly (docs/04-frontend.md).
      'no-restricted-globals': [
        'error',
        { name: 'fetch', message: 'Use HttpClient from @/shared/api.' },
      ],
    },
  },
  {
    files: ['src/shared/api/**', '**/*.test.{ts,tsx}'],
    rules: { 'no-restricted-globals': 'off' },
  },
);
