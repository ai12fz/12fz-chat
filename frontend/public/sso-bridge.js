/**
 * 12FZ SSO Bridge — 多系统统一登录脚本
 * 用法：各系统页面加载时执行此脚本，自动完成SSO认证
 *
 * 在模板中嵌入：
 *   window.CURRENT_USER = "username";           // 必填：用户名
 *   window.CURRENT_SOURCE = "erp";              // 必填：来源标识
 *   window.CURRENT_DISPLAY_NAME = "显示名";     // 选填：显示名称
 *   window.CHAT_SSO_SECRET = "密钥";            // 选填：默认从chat获取
 */
(function() {
  'use strict';

  const CHAT_BASE = 'https://chat.12fz.com';
  const STORAGE_KEY = 'chat_sso_token';
  const USER_KEY = 'chat_sso_user';

  // 等待 window.CURRENT_USER 就绪（部分系统模板在脚本后设置）
  function waitForUser(retries) {
    if (typeof window.CURRENT_USER !== 'undefined' && window.CURRENT_USER) {
      return Promise.resolve();
    }
    if (retries <= 0) return Promise.reject('CURRENT_USER not set');
    return new Promise(r => setTimeout(r, 200)).then(() => waitForUser(retries - 1));
  }

  // 解码JWT payload
  function parseJWT(token) {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) return null;
      const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
      return payload;
    } catch(e) {
      return null;
    }
  }

  // 检查已有token是否有效（未过期）
  function isValidToken() {
    const token = localStorage.getItem(STORAGE_KEY);
    if (!token) return null;
    const payload = parseJWT(token);
    if (!payload || Date.now() / 1000 > payload.exp) {
      localStorage.removeItem(STORAGE_KEY);
      localStorage.removeItem(USER_KEY);
      return null;
    }
    return token;
  }

  // 调用SSO登录
  function ssoLogin() {
    const source = window.CURRENT_SOURCE || 'unknown';
    const userId = window.CURRENT_USER;
    const displayName = window.CURRENT_DISPLAY_NAME || userId;

    return fetch(CHAT_BASE + '/api/sso/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        source: source,
        user_id: userId,
        display_name: displayName,
        sso_secret: '12fz-sso-2026'
      })
    })
    .then(r => {
      if (!r.ok) throw new Error('SSO login failed: ' + r.status);
      return r.json();
    })
    .then(data => {
      localStorage.setItem(STORAGE_KEY, data.token);
      localStorage.setItem(USER_KEY, JSON.stringify({
        bot_id: data.bot_id,
        source: data.source,
        name: data.source_name
      }));
      return data;
    });
  }

  // 获取未读消息总数
  function fetchUnreadCount() {
    const token = localStorage.getItem(STORAGE_KEY);
    if (!token) return Promise.resolve(0);

    return fetch(CHAT_BASE + '/api/messages/unread', {
      headers: { 'Authorization': 'Bearer ' + token }
    })
    .then(r => {
      if (!r.ok) throw new Error('unread api error');
      return r.json();
    })
    .then(data => {
      // data 格式: { "1": 5, "2": 3 }  group_id -> count
      let total = 0;
      for (let k in data) {
        if (data.hasOwnProperty(k)) total += data[k];
      }
      return total;
    })
    .catch(() => 0);
  }

  // 更新所有 .chat-badge 元素
  function updateBadges() {
    fetchUnreadCount().then(count => {
      document.querySelectorAll('.chat-badge').forEach(el => {
        if (count > 0) {
          el.textContent = count > 99 ? '99+' : count;
          el.style.display = '';
        } else {
          el.style.display = 'none';
        }
      });
    });
  }

  // 给信封图标加badge（兼容现有ERP/WP header结构）
  function initBadge() {
    // 查找现有的信封图标链接
    const envelopeLinks = document.querySelectorAll(
      'a[href*="envelope"], a[href*="mail"], a[href*="/index/vmain/main"], ' +
      'a i.fa-envelope-o, a i.fa-envelope, a[class*="message"], a[class*="chat"]'
    );

    envelopeLinks.forEach(el => {
      const link = el.tagName === 'A' ? el : el.closest('a');
      if (!link) return;

      // 改href指向chat
      link.href = CHAT_BASE + '/chat';

      // 已有badge则不重复加
      if (link.querySelector('.chat-badge')) return;

      // 创建badge
      const badge = document.createElement('span');
      badge.className = 'chat-badge';
      badge.style.cssText =
        'position:absolute;top:-6px;right:-6px;' +
        'background:#e74c3c;color:#fff;font-size:11px;' +
        'padding:1px 5px;border-radius:10px;' +
        'min-width:16px;text-align:center;' +
        'line-height:1.4;font-weight:bold;' +
        'display:none;z-index:999;';
      link.style.position = 'relative';
      link.appendChild(badge);
    });
  }

  // ── 主流程 ──
  waitForUser(10)
    .then(() => {
      const existing = isValidToken();
      if (existing) {
        // token有效，直接取未读
        initBadge();
        updateBadges();
        // 每60秒刷新未读
        setInterval(updateBadges, 60000);
        return;
      }
      // 无有效token，重新SSO登录
      return ssoLogin().then(() => {
        initBadge();
        updateBadges();
        setInterval(updateBadges, 60000);
      });
    })
    .catch(err => {
      console.log('[SSO] skip:', err);
    });
})();
