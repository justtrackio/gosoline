// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

const organizationName = "justtrackio";
const projectName = "gosoline";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Gosoline Docs',
  tagline: 'Dinosaurs are cool',
  favicon: 'img/logo-dark.png',

  // Set the production url of your site here
  url: `https://${organizationName}.github.io`,
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: `/${projectName}/`,

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName, // Usually your GitHub org/user name.
  projectName, // Usually your repo name.

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/justtrackio/gosoline/tree/main/docs/',
        },
        blog: {
          routeBasePath: '/blog',
          showReadingTime: true,
          blogTitle: 'Gosoline Blog',
          blogDescription: 'Articles and announcements from the gosoline project',
          editUrl: 'https://github.com/justtrackio/gosoline/tree/main/docs/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: 'img/gosoline-social-card.png',
      navbar: {
        title: 'Gosoline',
        logo: {
          alt: 'justtrack Logo',
          src: 'img/logo-transparent.png',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docSidebar',
            position: 'left',
            label: 'Docs',
          },
          {
            to: '/blog',
            label: 'Blog',
            position: 'left',
          },
          {
            href: 'https://github.com/justtrackio/gosoline',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      algolia: {
        appId: 'ER8OFD58GM',
        apiKey: '213984dd6a442d5a2401e35d1c8bb9f5', // This is a public, read-only key
        indexName: 'gosoline docs',
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Overview',
                to: '/',
              },
              {
                label: 'Quickstart tutorials',
                to: '/category/quickstart-tutorials',
              },
              {
                label: 'How-to guides',
                to: '/category/how-to-guides',
              },
              {
                label: 'Reference',
                to: '/category/reference',
              },
              {
                label: 'Blog',
                to: '/blog',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/justtrackio/gosoline',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} justtrack`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;
