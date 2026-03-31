import process from 'node:process';
import path from 'node:path';
import fs from 'node:fs';
import unocss from '@unocss/vite';
import presetIcons from '@unocss/preset-icons';
import { FileSystemIconLoader } from '@iconify/utils/lib/loader/node-loaders';

export function setupUnocss(viteEnv: Env.ImportMeta) {
  const { VITE_ICON_PREFIX, VITE_ICON_LOCAL_PREFIX } = viteEnv;

  const localIconPath = path.join(process.cwd(), 'src/assets/svg-icon');

  /** The name of the local icon collection */
  const collectionName = VITE_ICON_LOCAL_PREFIX.replace(`${VITE_ICON_PREFIX}-`, '');

  /**
   * Manual icons to be included in safelist These icons are used dynamically in the code and cannot be detected by
   * scanner
   */
  const manualIcons = [
    'ph:caret-double-left-bold',
    'ph:caret-double-right-bold',
    'mdi-pin-off',
    'mdi-pin',
    viteEnv.VITE_MENU_ICON || 'mdi:menu'
  ];

  const formatIconClass = (icon: string) => {
    return `${VITE_ICON_PREFIX}-${icon.replace(/:/g, '-')}`;
  };

  /** Scan all icons from source code pattern: "collection:icon" */
  const scanIcons = () => {
    const icons: string[] = [];

    try {
      const srcDir = path.join(process.cwd(), 'src');

      const scanFile = (filePath: string) => {
        const stat = fs.statSync(filePath);
        if (stat.isDirectory()) {
          fs.readdirSync(filePath).forEach(file => {
            scanFile(path.join(filePath, file));
          });
        } else if (/\.(vue|ts|tsx)$/.test(filePath)) {
          const content = fs.readFileSync(filePath, 'utf-8');
          const matches = content.matchAll(/['"]([\w-]+:[\w-]+)['"]/g);
          for (const match of matches) {
            icons.push(match[1]);
          }
        }
      };

      scanFile(srcDir);
    } catch (e) {
      console.warn('Failed to scan icons:', e);
    }

    return icons;
  };

  return unocss({
    presets: [
      presetIcons({
        prefix: `${VITE_ICON_PREFIX}-`,
        scale: 1,
        extraProperties: {
          display: 'inline-block'
        },
        collections: {
          [collectionName]: FileSystemIconLoader(localIconPath, svg =>
            svg.replace(/^<svg\s/, '<svg width="1em" height="1em" ')
          )
        },
        warn: true
      })
    ],
    // 自动扫描路由文件中的图标并添加到 safelist
    safelist: [...manualIcons.map(formatIconClass), ...scanIcons().map(formatIconClass)]
  });
}
