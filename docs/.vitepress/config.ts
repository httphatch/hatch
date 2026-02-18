import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Hatch',
  description: 'Local HTTPS development for macOS',
  cleanUrls: true,
  appearance: false,

  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/what-is-hatch' },
      { text: 'Reference', link: '/reference/cli' },
      { text: 'Advanced', link: '/advanced/troubleshooting' },
    ],

    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'What is Hatch?', link: '/guide/what-is-hatch' },
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Dashboard', link: '/guide/dashboard' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'CLI Commands', link: '/reference/cli' },
          { text: 'Configuration', link: '/reference/config' },
          { text: 'hatch.yml', link: '/reference/hatch-yml' },
        ],
      },
      {
        text: 'Advanced',
        items: [
          { text: 'Troubleshooting', link: '/advanced/troubleshooting' },
          { text: 'How It Works', link: '/advanced/how-it-works' },
        ],
      },
      {
        text: 'Community',
        items: [
          { text: 'Contributing', link: '/contributing' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/httphatch/hatch' },
    ],

    editLink: {
      pattern: 'https://github.com/httphatch/hatch/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },

    search: {
      provider: 'local',
    },

    footer: {
      message: 'A project by <a href="https://paulrose.com">Paul Rose</a>',
      copyright: 'Copyright © 2025-present Paul Rose',
    },
  },
})
