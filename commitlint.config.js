// Commitlint configuration for Conventional Commits
// https://commitlint.js.org

module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Type validation
    'type-enum': [
      2, // error
      'always',
      [
        'feat', // New feature (minor version bump)
        'fix', // Bug fix (patch version bump)
        'docs', // Documentation only
        'style', // Formatting, missing semi colons, etc
        'refactor', // Code change that neither fixes bug nor adds feature
        'perf', // Performance improvement
        'test', // Adding missing tests
        'chore', // Maintenance tasks (no version bump)
        'ci', // CI/CD changes
        'build', // Build system or dependencies
        'revert', // Revert previous commit
      ],
    ],

    // Scope validation (optional but recommended)
    'scope-enum': [
      2,
      'always',
      [
        'proto', // Protocol buffer definitions
        'schema', // JSON schema changes
        'sdk', // Go SDK changes
        'testing', // Testing framework
        'pricing', // Pricing package
        'registry', // Registry package
        'currency', // Currency package
        'pluginsdk', // Plugin SDK package
        'examples', // Example files
        'ci', // CI/CD pipeline
        'deps', // Dependencies
        'deps-dev', // Dev dependencies (used by Dependabot)
        'docs', // Documentation
        'main', // Main repository/release changes
      ],
    ],

    // Allow empty scope for general changes
    'scope-empty': [0],

    // Length constraints
    'header-max-length': [2, 'always', 72],
    'body-max-line-length': [2, 'always', 100],

    // Required elements
    'type-empty': [2, 'never'],
    'subject-empty': [2, 'never'],

    // Format rules
    'type-case': [2, 'always', 'lower-case'],
    // Disabled: technical subjects often contain PascalCase identifiers (e.g., BatchCost)
    'subject-case': [0],

    // No period at end of subject
    'subject-full-stop': [2, 'never', '.'],

    // Body should be separated by blank line
    'body-leading-blank': [2, 'always'],

    // Footer should be separated by blank line
    'footer-leading-blank': [2, 'always'],
  },
  ignores: [
    // Ignore WIP commits
    (message) => message.includes('WIP'),
    // Ignore merge commits
    (message) => message.includes('Merge branch'),
    (message) => message.includes('Merge pull request'),
  ],
};
