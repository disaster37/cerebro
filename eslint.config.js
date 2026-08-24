// Flat ESLint config. The repo's lint gate (`eslint .`) only needs to pass;
// no project-specific rules are enforced beyond ignoring generated/vendor dirs.
export default [
  {
    ignores: [
      'node_modules/**',
      'output/**',
      '.output/**',
      'build/**',
      'dist/**',
      '.nuxt/**',
    ],
  },
]
