import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import llmstxt from 'vitepress-plugin-llms'

// 环境变量驱动的 base path 配置（GitHub Pages 子路径）
const rawBase = process.env.VITEPRESS_BASE ?? (
  process.env.GITHUB_ACTIONS === 'true'
    ? `/${process.env.GITHUB_REPOSITORY?.split('/')[1] ?? 'chatroom'}/`
    : '/'
)
const base = rawBase.startsWith('/')
  ? rawBase.endsWith('/')
    ? rawBase
    : `${rawBase}/`
  : `/${rawBase}/`

const siteUrl = `https://holtwood-projects.github.io/chatroom/`

export default withMermaid(defineConfig({
  title: 'ChatRoom',
  description: '面向教学的实时聊天室',
  lang: 'zh-CN',
  base,
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: [/^http:\/\/localhost:\d+(?:\/.*)?$/],

  sitemap: {
    hostname: siteUrl,
  },

  head: [
    ['link', { rel: 'canonical', href: siteUrl }],
    ['meta', { name: 'theme-color', content: '#1e3a5f' }],
    ['meta', { name: 'author', content: 'holtwood-projects' }],
    ['meta', { name: 'keywords', content: 'ChatRoom, Go, React, WebSocket, PostgreSQL, 实时聊天, 教学项目' }],
    ['meta', { name: 'viewport', content: 'width=device-width, initial-scale=1.0' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'ChatRoom 教学文档' }],
    ['meta', { property: 'og:url', content: siteUrl }],
    ['meta', { property: 'og:image', content: `${siteUrl}og-image.png` }],
    ['meta', { property: 'og:image:width', content: '1200' }],
    ['meta', { property: 'og:image:height', content: '630' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:image', content: `${siteUrl}og-image.png` }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: `${base}logo.svg` }],
    ['link', { rel: 'icon', type: 'image/png', href: `${base}favicon.png` }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: `${base}apple-touch-icon.png` }],
    ['link', { rel: 'manifest', href: `${base}manifest.json` }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap' }],
  ],

  markdown: {
    lineNumbers: true,
    theme: { light: 'github-light', dark: 'github-dark' },
  },

  mermaid: {
    startOnLoad: true,
    theme: 'default',
  },

  vite: {
    plugins: [llmstxt()],
  },

  themeConfig: {
    siteTitle: 'ChatRoom',
    logo: '/logo.svg',
    socialLinks: [
      { icon: 'github', link: 'https://github.com/holtwood-projects/chatroom' },
    ],
    externalLinkIcon: true,
    editLink: {
      pattern: 'https://github.com/holtwood-projects/chatroom/edit/main/docs/:path',
      text: '在 GitHub 上编辑此页',
    },
    nav: [
      { text: '架构', link: '/architecture/system', activeMatch: '/architecture/' },
      { text: 'API', link: '/api/rest', activeMatch: '/api/' },
      { text: '教程', link: '/tutorials/local-dev', activeMatch: '/tutorials/' },
    ],
    sidebar: [
      {
        text: '架构',
        collapsed: false,
        items: [
          { text: '系统架构', link: '/architecture/system' },
          { text: '数据流', link: '/architecture/data-flow' },
          { text: '数据模型', link: '/architecture/data-model' },
        ],
      },
      {
        text: 'API 参考',
        collapsed: false,
        items: [
          { text: 'REST API', link: '/api/rest' },
          { text: 'WebSocket 协议', link: '/api/websocket' },
        ],
      },
      {
        text: '教程',
        collapsed: false,
        items: [
          { text: '本地开发', link: '/tutorials/local-dev' },
          { text: '测试指南', link: '/tutorials/testing' },
        ],
      },
      {
        text: '参考',
        collapsed: true,
        items: [
          { text: '常见问题', link: '/reference/faq' },
          { text: '变更日志', link: '/reference/changelog' },
        ],
      },
    ],
    outline: { label: '本页目录' },
    docFooter: { prev: '上一页', next: '下一页' },
    footer: {
      message: '基于 MIT 许可证发布',
      copyright: 'Copyright © 2025-2026 holtwood-projects',
    },
    lastUpdated: { text: '最后更新于' },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
    darkModeSwitchLabel: '主题',
    lightModeSwitchTitle: '切换到浅色模式',
    darkModeSwitchTitle: '切换到深色模式',
    search: {
      provider: 'local',
      options: {
        translations: {
          button: {
            buttonText: '搜索文档',
            buttonAriaLabel: '搜索文档',
          },
          modal: {
            noResultsText: '无法找到相关结果',
            resetButtonTitle: '清除查询条件',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
            },
          },
        },
      },
    },
  },
}))
