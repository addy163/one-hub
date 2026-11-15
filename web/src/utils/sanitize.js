/**
 * 轻量 HTML 白名单消毒（不引入第三方依赖）。
 *
 * 用于渲染后台可编辑的公告 / 关于 / 首页内容 / 页脚 HTML。
 * 这些内容是公开接口返回的，且管理员账号一旦失陷即可写入任意脚本；
 * 原实现直接 marked + dangerouslySetInnerHTML，构成存储型 XSS，
 * 攻击者可借此在受害者浏览器里以登录态调用 /api/token/ 拿走密钥。
 */

const ALLOWED_TAGS = new Set([
  'a', 'p', 'br', 'hr', 'div', 'span', 'strong', 'b', 'em', 'i', 'u', 's', 'del', 'ins',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code',
  'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'caption', 'colgroup', 'col',
  'img', 'figure', 'figcaption', 'sup', 'sub', 'mark', 'details', 'summary', 'center', 'small'
]);

// 直接整棵子树删除的标签（保留文本对排查无意义，且 script/style 文本也可被误解析）
const DROP_SUBTREE = new Set([
  'script', 'style', 'iframe', 'frame', 'frameset', 'object', 'embed', 'applet',
  'link', 'meta', 'base', 'form', 'input', 'button', 'textarea', 'select', 'option',
  'svg', 'math', 'template', 'noscript', 'audio', 'video', 'source', 'track'
]);

const ALLOWED_ATTRS = {
  a: new Set(['href', 'title', 'target', 'rel']),
  img: new Set(['src', 'alt', 'title', 'width', 'height']),
  '*': new Set(['class', 'title', 'dir', 'lang'])
};

function isUnsafeUrl(value) {
  const v = String(value || '').trim().replace(/[\u0000-\u001f]/g, '').toLowerCase();
  return v.startsWith('javascript:') || v.startsWith('data:') || v.startsWith('vbscript:') || v.startsWith('file:');
}

export function sanitizeHtml(dirty) {
  if (dirty === null || dirty === undefined) return '';
  const html = String(dirty);
  if (typeof window === 'undefined' || typeof DOMParser === 'undefined') {
    // SSR 环境下不渲染任何后台 HTML，避免未经消毒的内容进入 DOM
    return '';
  }
  let doc;
  try {
    doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html');
  } catch (e) {
    return '';
  }
  const body = doc.body;
  if (!body) return '';

  const walk = (node) => {
    const children = Array.from(node.childNodes || []);
    children.forEach((child) => {
      if (child.nodeType === 8) {
        // 注释节点：IE 条件注释等历史攻击面，直接移除
        child.remove();
        return;
      }
      if (child.nodeType !== 1) return;

      const tag = child.tagName.toLowerCase();
      if (DROP_SUBTREE.has(tag) || !ALLOWED_TAGS.has(tag)) {
        child.remove();
        return;
      }

      Array.from(child.attributes || []).forEach((attr) => {
        const name = attr.name.toLowerCase();
        const allowed = ALLOWED_ATTRS[tag] || new Set();
        const permitted = allowed.has(name) || ALLOWED_ATTRS['*'].has(name);
        if (!permitted || name.startsWith('on') || isUnsafeUrl(attr.value) && (name === 'href' || name === 'src')) {
          child.removeAttribute(attr.name);
          return;
        }
        if (name === 'href' || name === 'src') {
          if (isUnsafeUrl(attr.value)) child.removeAttribute(attr.name);
        }
      });

      // 外链补充 noopener，避免 tabnabbing
      if (tag === 'a' && child.getAttribute('target') === '_blank') {
        const rel = (child.getAttribute('rel') || '').split(/\s+/);
        if (!rel.includes('noopener')) rel.push('noopener');
        if (!rel.includes('noreferrer')) rel.push('noreferrer');
        child.setAttribute('rel', rel.filter(Boolean).join(' '));
      }

      walk(child);
    });
  };

  walk(body);
  return body.innerHTML;
}

export default sanitizeHtml;
